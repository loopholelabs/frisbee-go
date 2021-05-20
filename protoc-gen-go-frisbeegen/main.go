package main

import (
	"github.com/loophole-labs/frisbee/internal/generator"
	"google.golang.org/protobuf/compiler/protogen"
)

func main() {
	protogen.Options{}.Run(func(gen *protogen.Plugin) error {
		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			fbg := generator.New(gen, f)
			fbg.GenerateFrisbeeFiles()
		}
		return nil
	})
}
