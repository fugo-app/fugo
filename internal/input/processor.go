package input

type Processor interface {
	Process(data map[string]string)
}
