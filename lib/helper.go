package lib

import (
	"os"
)

// WriteTempJS writes code to disk so you can import it via Node.
// It always generates a file with a ".js" extension.
// Note that cleanup can be called immediately after Node has run once, as once Node imports a file once, it'll use its cache to `import` it again.
func WriteTempJS(name, code string) (p string, cleanup func(), err error) {
	return writeTemp(name, code, "js")
}

func writeTemp(name, code, suffix string) (p string, cleanup func(), err error) {
	f, err := os.CreateTemp(os.TempDir(), name+".*."+suffix)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()

	_, err = f.Write([]byte(code))
	if err != nil {
		os.Remove(p)
		return
	}

	p = f.Name()
	cleanup = func() { os.Remove(p) }
	return
}
