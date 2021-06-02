package inventory

import (
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/disintegration/imaging"
	"github.com/edwvee/exiffix"
	"gopkg.in/yaml.v2"
)

const (
	itemYAML   = "info.yaml"
	itemPic    = "picture.jpg"
	itemLocPic = "location.jpg"
	retCODE    = "RETURN_CODE"
)

// Item is the item in the inventory.
type Item struct {
	ID       string    `yaml:"id"`
	SKU      string    `yaml:"sku"`
	Name     string    `yaml:"name"`
	Type     string    `yaml:"itemtype"`
	Value    string    `yaml:"value"`
	Size     string    `yaml:"size"`
	Quantity string    `yaml:"quantity"`
	Price    string    `yaml:"price"`
	Location string    `yaml:"location"`
	Updated  time.Time `yaml:"update"`
}

// Delete deletes the item from the disk
func (i *Item) Delete() error {
	err := os.RemoveAll(i.path(""))
	if err != nil {
		return fmt.Errorf("inventory: could not delete item directory: %w", err)
	}

	return nil
}

// Update updates the information of the item on disk.
func (i *Item) Update() error {
	i.Updated = time.Now()

	err := os.MkdirAll(i.path(""), os.ModePerm)
	if err != nil {
		return fmt.Errorf("inventory: could not create item directory: %w", err)
	}

	data, err := yaml.Marshal(i)
	if err != nil {
		return fmt.Errorf("inventory: could not marshal yaml file: %w", err)
	}
	ioutil.WriteFile(i.path(itemYAML), data, 0644)
	return nil
}

// SetPicture sets the thumbnail picture of the item. To save space, the image
// is resized within 1000x1000 pixels and encoded as jpeg.
func (i *Item) SetPicture(r io.ReadSeeker) error {
	return parseImg(r, 1000, i.path(itemPic))
}

// SetLocationPicture sets the thumbnail picture of the item. To save space, the image
// is resized within 1000x1000 pixels and encoded as jpeg.
func (i *Item) SetLocationPicture(r io.ReadSeeker) error {
	return parseImg(r, 1500, i.path(itemLocPic))
}

func parseImg(r io.ReadSeeker, size int, filepath string) error {
	data, _, err := exiffix.Decode(r)
	if err != nil {
		return fmt.Errorf("inventory: could not decode image: %w", err)
	}

	img := imaging.Thumbnail(data, size, size, imaging.Lanczos)

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("inventory: could not create image file: %w", err)
	}
	defer file.Close()
	err = jpeg.Encode(file, img, nil)
	if err != nil {
		return fmt.Errorf("inventory: could not encode image file: %w", err)
	}

	return nil
}

// Picture returns the picture associated with the item.
func (i *Item) Picture() (image.Image, error) {
	return getImg(i.path(itemPic))
}

// LocationPicture returns the picture of the location associated with the item.
func (i *Item) LocationPicture() (image.Image, error) {
	return getImg(i.path(itemLocPic))
}

// String implements the Stringer interface.
func (i *Item) String() string {
	return fmt.Sprintf("{%s (%s) at %s, InUse: %v, Updated: %v}", i.ID, i.Name, i.Location, i.InUse, i.Updated)
}

func getImg(filepath string) (image.Image, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("inventory: could not open image file: %w", err)
	}
	defer file.Close()

	img, err := jpeg.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("inventory: could not decode image: %w", err)
	}

	return img, nil
}

func (i *Item) path(filename string) string {
	if filename == "" {
		return filepath.Join(getDir(), i.ID)
	}
	return filepath.Join(getDir(), i.ID, filename)
}

// by is the type of the `Less` function.
type by func(i1, i2 *Item) bool

// sorter sorts a list of items by a specific variable.
func (b by) sorter(items []*Item, reversed bool) {
	is := &itemSorter{
		items: items,
		by:    b,
	}

	if reversed {
		sort.Sort(sort.Reverse(is))
	} else {
		sort.Sort(is)
	}
}

type itemSorter struct {
	items []*Item
	by    func(i1, i2 *Item) bool
}

func (is *itemSorter) Len() int {
	return len(is.items)
}

func (is *itemSorter) Swap(i, j int) {
	is.items[i], is.items[j] = is.items[j], is.items[i]
}

func (is *itemSorter) Less(i, j int) bool {
	return is.by(is.items[i], is.items[j])
}

type sortBy byte

const (
	// ByName sorts items by name.
	ByName sortBy = (1 << iota)
	// ByDate sorts items by update date.
	ByDate
	// ByPrice sorts items by price.
	ByPrice
)

// Sort sorts a slice of items by the speciafied element.
func Sort(element sortBy, items []*Item, reversed bool) {
	switch element {
	case ByName:
		by(func(i1, i2 *Item) bool {
			return i1.Name < i2.Name
		}).sorter(items, reversed)
	case ByDate:
		by(func(i1, i2 *Item) bool {
			return i1.Updated.Before(i2.Updated)
		}).sorter(items, reversed)
	case ByPrice:
		by(func(i1, i2 *Item) bool {
			return i1.Price < i2.Price
		}).sorter(items, reversed)
	}
}
