package utils

func Invariant(isPassed bool, message string) {
	if !isPassed {
		panic(message)
	}
}
