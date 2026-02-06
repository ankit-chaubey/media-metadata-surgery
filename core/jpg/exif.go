package jpg

import (
	"fmt"
	"os"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
)

func ViewEXIF(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return fmt.Errorf("no EXIF metadata found")
	}

	fmt.Println("EXIF Metadata:")
	x.Walk(walker{})
	return nil
}

type walker struct{}

func (w walker) Walk(name exif.FieldName, tag *tiff.Tag) error {
	fmt.Printf("%s: %v\n", name, tag)
	return nil
}
