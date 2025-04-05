package file

type fileParser interface {
	Parse(line string) (map[string]string, error)
}
