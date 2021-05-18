package main

import (
	"google.golang.org/protobuf/compiler/protogen"
	"protoc-gen-go-frisbeegen/internal/frisbeegenerator"
)

func main() {
	protogen.Options{}.Run(func(gen *protogen.Plugin) error {
		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			fbg := frisbeegenerator.New(gen, f)
			fbg.GenerateFrisbeeFiles()
		}
		return nil
	})
}
