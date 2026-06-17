package action

import "os"

func osEnviron() []string { return os.Environ() }
