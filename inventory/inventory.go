package inventory

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v2"
)

var (
	// CustomPath sets a custom directory to store the inventory.
	CustomPath string
	// ReturnLocation sets the return location of the inventory (default:
	// returned).
	ReturnLocation = "returned"
)

// Items returns the list of items in the inventory.
func Items() ([]*Item, error) {
	path := getDir()
	itemsDirs, err := ioutil.ReadDir(path)
	if os.IsNotExist(err) {
		fmt.Println("looks like", path, "does not exist, I will create it")
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return nil, fmt.Errorf("inventory: could not create directory: %w", err)
		}
		fmt.Println("done!")
	} else if err != nil {
		return nil, fmt.Errorf("inventory: could not read directory: %w", err)
	}

	var readers sync.WaitGroup
	var managers sync.WaitGroup

	items := []*Item{}
	queue := make(chan *Item)
	errors := make(chan error)

	managers.Add(1)
	go func() {
		defer managers.Done()
		for i := range queue {
			items = append(items, i)
		}
	}()

	managers.Add(1)
	go func() {
		defer managers.Done()
		for e := range errors {
			err = fmt.Errorf("%v\n%w", err, e)
		}
	}()

	readers.Add(len(itemsDirs))
	for _, dir := range itemsDirs {
		dir := dir
		go func() {
			defer readers.Done()
			if dir.IsDir() {
				yamlFile := filepath.Join(path, dir.Name(), itemYAML)
				data, err := ioutil.ReadFile(yamlFile)
				if err != nil {
					errors <- fmt.Errorf("inventory: could not parse item: %w", err)
					return
				}
				var i Item
				if err := yaml.Unmarshal(data, &i); err != nil {
					errors <- fmt.Errorf("inventory: could not parse item: %w", err)
					return
				}
				queue <- &i
			}
		}()
	}

	readers.Wait()
	close(queue)
	close(errors)
	managers.Wait()

	return items, err
}

// SortedItems returns a sorted slice of items in the inventory.
func SortedItems(by sortBy, reversed bool) ([]*Item, error) {
	items, err := Items()
	if err != nil {
		return nil, err
	}
	Sort(by, items, reversed)

	return items, nil
}

// Add adds a new named item to the inventory. It will auto-generate a unique
// ID for the item based on the name.
func Add(sku, name, itemtype, value, size, quantity, price, location string) (*Item, error) {
	item := &Item{
		ID:       uniqueKey(name),
		SKU:      sku,
		Name:     name,
		Type:     itemtype,
		Value:    value,
		Size:     size,
		Quantity: quantity,
		Price:    price,
		Location: location,
	}

	img, err := base64.StdEncoding.DecodeString(imgDEFAULT)
	if err != nil {
		return nil, fmt.Errorf("inventory: could not decode default image: %w", err)
	}
	err = item.Update()
	if err != nil {
		return nil, fmt.Errorf("inventory: could not add item: %w", err)
	}
	err = item.SetPicture(bytes.NewReader(img))
	if err != nil {
		return nil, fmt.Errorf("inventory: could not add item: %w", err)
	}
	err = item.SetLocationPicture(bytes.NewReader(img))
	if err != nil {
		return nil, fmt.Errorf("inventory: could not add item: %w", err)
	}

	return item, nil
}

// Update updates an item in the inventory by ID.
func Update(id, sku, name, itemtype, value, size, quantity, price, location string) (*Item, error) {
	item := &Item{
		ID:       id,
		SKU:      sku,
		Name:     name,
		Type:     itemtype,
		Value:    value,
		Size:     size,
		Quantity: quantity,
		Price:    price,
		Location: location,
	}

	err := item.Update()
	if err != nil {
		return nil, fmt.Errorf("inventory: could not add item: %w", err)
	}

	return item, nil
}

// Delete deletes an item from the inventory.
func Delete(id string) (error) {
	item := &Item{
		ID: id,
	}

	err := item.Delete()
	if err != nil {
		return fmt.Errorf("inventory: could not delete item: %w", err)
	}
	return err
}

func getDir() string {
	if CustomPath != "" {
		CustomPath += "/inventory"
		return CustomPath
	}
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	return filepath.Join(home, ".warehouse/inventory")
}

// Path returns the path of the inventory.
func Path() string {
	return getDir()
}

func uniqueKey(key string) string {
	path := getDir()
	mark := 'a'
	key = fmt.Sprintf("%.10s", clean(key))
	valid := key

	for _, err := os.Stat(filepath.Join(path, valid)); !os.IsNotExist(err); _, err = os.Stat(filepath.Join(path, valid)) {
		valid = fmt.Sprintf("%s_%s", key, string(mark))
		mark++
	}

	return valid
}

func clean(s string) string {
	rx, err := regexp.Compile("[^[:alnum:][:space:]]+")
	if err != nil {
		return s
	}

	s = rx.ReplaceAllString(s, " ")
	s = strings.Replace(s, " ", "", -1)

	return strings.ToLower(s)
}
