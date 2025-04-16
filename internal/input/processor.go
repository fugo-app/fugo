package input

type Processor interface {
	// Serialize converts raw data to a structured data
	Serialize(data map[string]string) map[string]any

	// Write writes structured data to the storage
	Write(data map[string]any)
}
