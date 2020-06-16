package main

import (
	"archive/zip"
	"crypto/x509/pkix"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func userForm(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`
<html>
<body>
<h3>Demo Kafka Auth Generation</h3>
<div>
  <p>Create a new user:</p>
  <form method=POST action="/generate">
  <div style="padding: 1em">
    <label for="username">Username:</label>
    <input type="text" name="username">
  </div>
  <div style="padding: 1em">
    <label for="email">Email:</label>
    <input type="text" name="email">
  </div>
  <div style="padding: 1em">
    <input type="submit" value="Create">
  </div>
  </form>
</div>
</body>
</html>
`))
}

func generateCreds(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(fmt.Sprintf("unable to parse form: %v", err)))
		return
	}
	values := r.PostForm
	username := values.Get("username")
	password := randString(32)

	privateKey, err := runOpenSSL(
		"genrsa",
		"-des3",
		"-passout", "pass:"+password,
	)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("unable to generate private key: %v", err)))
		return
	}

	cert, err := generateCert(username, privateKey, password)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("unable to generate cert: %v", err)))
		return
	}

	bundleID, err := makeBundle(username, password, privateKey, cert)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("unable to generate bundle: %v", err)))
		return
	}

	w.Write([]byte(fmt.Sprintf(`
<html>
<body>
<h3><a href="/bundle?id=%s" download="%s-credentials.zip" target="_blank">One-time-use link to download credentials</a></h3>
<h3>Username:</h3>
<p>%s</p>
<h3>Config file sample:</h3>
<pre>%s</pre>
<h3>Public certificate:</h3>
<pre>%s</pre>
<h3>Private key password:</h3>
<pre>%s</pre>
<h3>Private key:</h3>
<pre>%s</pre>
</body>
</html>
`, bundleID, username, username, config(username, password), cert, password, privateKey)))
}

func config(username, privateKeyPassword string) string {
	return fmt.Sprintf(`
security.protocol=ssl
ssl.certificate.location=%s-public.pem
ssl.key.location=%s-private.pem
ssl.key.password=%s
`, username, username, privateKeyPassword)
}

func generateCert(username string, privateKey []byte, privateKeyPassword string) ([]byte, error) {
	tf, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tf.Name())
	_, err = tf.Write(privateKey)
	if err != nil {
		return nil, err
	}
	err = tf.Close()

	name := pkix.Name{
		Country:            []string{"US"},
		Organization:       []string{"SCIMMA"},
		OrganizationalUnit: []string{},
		Locality:           []string{"Seattle"},
		Province:           []string{"WA"},
		CommonName:         username,
	}

	return runOpenSSL(
		"req",
		"-key", tf.Name(),
		"-passin", "pass:"+privateKeyPassword,
		"-new",
		"-nameopt", "RFC2253",
		"-subj", openSSLFormatName(name),
		"-x509",
		"-days", "36500",
		"-outform", "pem",
	)
}

func randString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func openSSLFormatName(name pkix.Name) string {
	// HACK: This is completely wrong with internal commas, oh well
	return "/" + strings.Replace(name.String(), ",", "/", -1) + "/"
}

func runOpenSSL(args ...string) (output []byte, err error) {
	tempfile := "f.tmp"
	args = append(args, "-out", tempfile)
	cmd := exec.Command("openssl", args...)
	_, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error running openssl: %w\nstderr: %s", err, err.(*exec.ExitError).Stderr)
	}

	f, err := os.Open(tempfile)
	if err != nil {
		return nil, fmt.Errorf("error opening tempfile result: %w", err)
	}

	contents, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("error reading tempfile result: %w", err)
	}

	err = f.Close()
	if err != nil {
		return nil, fmt.Errorf("error closing tempfile result: %w", err)
	}

	err = os.Remove(tempfile)
	if err != nil {
		return nil, fmt.Errorf("error cleaning up tempfile result: %w", err)
	}
	return contents, nil
}

func makeBundle(username, password string, privateKey, cert []byte) (string, error) {
	id := username + "-" + randString(20) + ".zip"
	f, err := os.Create(id)
	if err != nil {
		return "", err
	}
	defer f.Close()
	zipW := zip.NewWriter(f)
	configFile, err := zipW.Create("client-" + username + ".config")
	if err != nil {
		return "", err
	}
	_, err = configFile.Write([]byte(config(username, password)))
	if err != nil {
		return "", err
	}

	privateKeyFile, err := zipW.Create(username + "-private.pem")
	if err != nil {
		return "", err
	}
	_, err = privateKeyFile.Write(privateKey)
	if err != nil {
		return "", err
	}
	certFile, err := zipW.Create(username + "-public.pem")
	if err != nil {
		return "", err
	}
	_, err = certFile.Write(cert)
	if err != nil {
		return "", err
	}

	err = zipW.Close()
	if err != nil {
		return "", err
	}
	return id, nil
}

func bundle(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	f, err := os.Open(id)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(fmt.Sprintf("unable to open bundle: %v", err)))
		return
	}
	defer f.Close()
	defer os.Remove(id)

	body, err := ioutil.ReadAll(f)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("unable to read bundle: %v", err)))
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Write(body)
}

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(userForm))
	mux.Handle("/generate", http.HandlerFunc(generateCreds))
	mux.Handle("/bundle", http.HandlerFunc(bundle))
	http.ListenAndServe(":8080", mux)
}
