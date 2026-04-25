package gsc

type Command struct {
	Name string
	Args []string
}

type CommandSet struct {
	Commands []Command
}

type Stream struct {
	Num    uint16
	Lang   uint8
	Ver    uint8
	CmdSet CommandSet
}
