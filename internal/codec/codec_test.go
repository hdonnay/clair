package codec

import (
	"fmt"
	"os"
)

func Example() {
	enc := GetEncoder(os.Stdout)
	defer PutEncoder(enc)
	enc.MustEncode([]string{"a", "slice", "of", "strings"})
	fmt.Fprintln(os.Stdout)
	enc.MustEncode(nil)
	fmt.Fprintln(os.Stdout)
	enc.MustEncode(map[string]string{})
	fmt.Fprintln(os.Stdout)
	// Output: ["a","slice","of","strings"]
	// null
	// {}
}
