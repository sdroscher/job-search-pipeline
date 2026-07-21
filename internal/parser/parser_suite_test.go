package parser_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// ParserSuite groups all parser tests that share the mock-server pattern.
type ParserSuite struct {
	suite.Suite
}

func TestParserSuite(t *testing.T) {
	suite.Run(t, new(ParserSuite))
}
