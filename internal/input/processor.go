package input

type Processor interface {
	// Process convert raw data to a structured data and writes it to the storage
	Process(data map[string]string)
	// Write writes structured data to the storage
	Write(data map[string]any)
}
