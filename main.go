package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"html/template"

	"github.com/medoix/warehouse/inventory"
	"github.com/medoix/warehouse/equipment"
	"github.com/markbates/pkger"
	"github.com/mitchellh/go-homedir"
	"github.com/skip2/go-qrcode"
)

var templates *template.Template

func main() {
	port := flag.Int("p", 8080, "port to serve the inventory")
	path := flag.String("d", defaultPath(), "path to warehouse directory")
	flag.Parse()

	if *path != defaultPath() {
		if _, err := os.Stat(*path); os.IsNotExist(err) {
			log.Fatalf("error with warehouse path: %v", err)
		}
		equipment.CustomPath = *path
		inventory.CustomPath = *path
	}

	equipment.Items()
	inventory.Items()

	f, err := os.OpenFile(filepath.Join(*path, "log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	templates, err = initTemplates(pkger.Include("/templates"))
	if err != nil {
		log.Fatalf("error reading templates: %v", err)
	}

	// Dashbaord routes
	http.HandleFunc("/", dashboardIndex)

	// Equipment static content like images
	http.Handle("/equipment/", http.StripPrefix("/equipment/", http.FileServer(http.Dir(inventory.Path()))))
	// Equipment routes for actions
	http.HandleFunc("/equipment/edit", equipmentEdit)
	http.HandleFunc("/equipment/qr", equipmentQr)
	http.HandleFunc("/equipment/location", equipmentLocation)
	http.HandleFunc("/equipment/add", equipmentAdd)
	http.HandleFunc("/equipment", equipmentIndex)

	http.Handle("/inventory/", http.StripPrefix("/inventory/", http.FileServer(http.Dir(inventory.Path()))))
	http.HandleFunc("/inventory/delete", inventoryDelete)
	http.HandleFunc("/inventory/edit", inventoryEdit)
	http.HandleFunc("/inventory/qr", inventoryQr)
	http.HandleFunc("/inventory/location", inventoryLocation)
	http.HandleFunc("/inventory/add", inventoryAdd)
	http.HandleFunc("/inventory", inventoryIndex)

	fmt.Printf("warehouse server started on port %d\n", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}

func defaultPath() string {
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	return filepath.Join(home, ".warehouse")
}

func initTemplates(dir string) (*template.Template, error) {
	t := template.New("")

	err := pkger.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}
		f, err := pkger.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		data, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}

		_, err = t.Parse(string(data))
		if err != nil {
			return err
		}

		return nil
	})

	return t, err
}

// Dashboard Functions
func dashboardIndex(w http.ResponseWriter, r *http.Request) {
	if err := templates.ExecuteTemplate(w, "dashboard",
		&struct {
			Title string
		}{
			Title: "Dashboard",
		},
	); err != nil {
		log.Println("[ERR]", err)
		return
	}
}

// Equipment Functions
func equipmentEdit(w http.ResponseWriter, r *http.Request) {
}

func equipmentQr(w http.ResponseWriter, r *http.Request) {
}

func equipmentLocation(w http.ResponseWriter, r *http.Request) {
}

func equipmentAdd(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		name := r.FormValue("name")
		item, err := equipment.Add(name)
		if err != nil {
			log.Println("[ERR]", err)
			return
		}

		img, _, err := r.FormFile("image")
		if err != nil {
			log.Println("[ERR]", err)
			return
		}
		defer img.Close()
		if err := item.SetPicture(img); err != nil {
			log.Println("[ERR]", err)
			return
		}

		log.Println("[ADD]", item)
		http.Redirect(w, r, "/equipment", http.StatusSeeOther)

	case "GET":
		if err := templates.ExecuteTemplate(w, "equipment-add", nil); err != nil {
			log.Println("[ERR]", err)
			return
		}
	}
}

func equipmentIndex(w http.ResponseWriter, r *http.Request) {
	items, err := equipment.SortedItems(equipment.ByInUseDate, true)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	if err := templates.ExecuteTemplate(w, "equipment",
		&struct {
			Title string
			Items []*equipment.Item
		}{
			Title: "Equipment",
			Items: items,
		},
	); err != nil {
		log.Println("[ERR]", err)
		return
	}
}

// Inventory Functions
func inventoryIndex(w http.ResponseWriter, r *http.Request) {
	items, err := inventory.SortedItems(inventory.ByPrice, true)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	if err := templates.ExecuteTemplate(w, "inventory",
		&struct {
			Title string
			Items []*inventory.Item
		}{
			Title: "Inventory",
			Items: items,
		},
	); err != nil {
		log.Println("[ERR]", err)
		return
	}
}

func inventoryAdd(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		sku      := r.FormValue("sku")
		name     := r.FormValue("name")
		itemtype := r.FormValue("type")
		value    := r.FormValue("value")
		size     := r.FormValue("size")
		quantity := r.FormValue("quantity")
		price    := r.FormValue("price")
		location := r.FormValue("location")
		item, err := inventory.Add(sku, name, itemtype, value, size, quantity, price, location)
		if err != nil {
			log.Println("[ERR]", err)
			return
		}

		img, _, err := r.FormFile("image")
		if err != nil {
			log.Println("[ERR]", err)
			return
		}
		defer img.Close()
		if err := item.SetPicture(img); err != nil {
			log.Println("[ERR]", err)
			return
		}

		log.Println("[ADD]", item)
		http.Redirect(w, r, "/inventory", http.StatusSeeOther)

	case "GET":
		if err := templates.ExecuteTemplate(w, "inventory-add", nil); err != nil {
			log.Println("[ERR]", err)
			return
		}
	}
}

func inventoryQr(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	qr, err := qrcode.Encode(fmt.Sprintf("http://%s/inventory/update?id=%s", r.Host, id), qrcode.Medium, 256)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	io.Copy(w, bytes.NewReader(qr))
}

func inventoryDelete(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		http.Redirect(w, r, "/inventory", http.StatusSeeOther)
		return
	}

	err := inventory.Delete(id)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	http.Redirect(w, r, "/inventory", http.StatusSeeOther)
}

func inventoryLocation(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	if id == "" {
		http.Redirect(w, r, "/inventory", http.StatusSeeOther)
		return
	}

	items, err := inventory.Items()
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	for _, item := range items {
		if item.ID == id {
			img, err := item.LocationPicture()
			if err != nil {
				log.Println("[ERR]", err)
				return
			}
			jpeg.Encode(w, img, nil)
			break
		}
	}
}

func inventoryEdit(w http.ResponseWriter, r *http.Request) {
	id       := r.FormValue("id")
	sku      := r.FormValue("sku")
	name     := r.FormValue("name")
	itemtype := r.FormValue("type")
	value    := r.FormValue("value")
	size     := r.FormValue("size")
	quantity := r.FormValue("quantity")
	price    := r.FormValue("price")
	location := r.FormValue("location")
	filename := r.FormValue("filename")

	if id == "" {
		http.Redirect(w, r, "/inventory", http.StatusSeeOther)
		return
	}

	items, err := inventory.Items()
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	for _, item := range items {
		if item.ID == id {
			switch r.Method {
			case "POST":
				inventory.Update(
					id,
					sku,
					name,
					itemtype,
					value,
					size,
					quantity,
					price,
					location,
				)

				if(filename != ""){
					img, _, err := r.FormFile("image")
					if err != nil {
						log.Println("[ERR]", err)
						return
					}
					defer img.Close()
					if err := item.SetPicture(img); err != nil {
						log.Println("[ERR]", err)
						return
					}
				}
				http.Redirect(w, r, "/inventory", http.StatusSeeOther)

			case "GET":
				if err := templates.ExecuteTemplate(w, "inventory-edit",
					&struct {
						Title string
						Item  *inventory.Item
					}{
						Title: item.Name,
						Item:  item,
					},
				); err != nil {
					log.Println("[ERR]", err)
					return
				}
			}
		}
	}
}
