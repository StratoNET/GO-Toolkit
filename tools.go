package toolkit

import (
	"crypto/rand"
	"log"
)

const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+-="

// Tools is used to instantiate this module. Any variable of this type will have access to all methods with the receiver *Tools
type Tools struct{}

// RandomString returns string of random characters of length n, generated from randomStringSource
func (t *Tools) RandomString(n int) string {
	s, r := make([]rune, n), []rune(randomStringSource)
	for i := range s {
		p, err := rand.Prime(rand.Reader, len(r))
		if err != nil {
			log.Println(err)
		}
		x, y := p.Uint64(), uint64(len(r))
		s[i] = r[x%y]
	}

	return string(s)
}
