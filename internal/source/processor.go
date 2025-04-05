package source

type Processor interface {
	Process(data map[string]string)
}
