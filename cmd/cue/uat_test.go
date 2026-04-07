package main

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type UATCommandSuite struct {
	suite.Suite
}

func TestUATCommand(t *testing.T) {
	suite.Run(t, new(UATCommandSuite))
}

func (s *UATCommandSuite) TestUATCommandReturnsValidCommand() {
	cmd := uatCommand()
	s.NotNil(cmd)
	s.Equal("uat", cmd.Name)
	s.Equal("Launch character UAT mode (full UI, no services)", cmd.Usage)
	s.NotNil(cmd.Action)
}
