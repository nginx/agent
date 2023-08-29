package worker

type MetaData struct {
	Type            string
	LocalId         string
	Uuid            string
	Name            string
	DisplayName     string
	Listen          string
	Flisten         string
	StatusPath      string
	CanHaveChildren bool
	Agent           string
	ParentLocalId   string
	Includes        []string
}
