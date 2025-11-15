package main

import (
	"io"
)

type Plug interface {
	TurnOn() (on bool, err error)
	TurnOff() (off bool, err error)
	RenderEnergy(io.Writer) error
	RenderInfo(io.Writer) error
}
