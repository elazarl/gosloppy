package pkg

import "libtest/stam"
import "os"

func g() { stam.F() }

func F(a int) { var b int; c := 1; os.Stdout.WriteString("SUCCESS") }
