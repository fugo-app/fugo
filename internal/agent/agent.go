package agent

type Agent struct {
	Name string     `yaml:"name"`
	File *FileAgent `yaml:"file,omitempty"`
}
