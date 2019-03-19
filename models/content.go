package models

import (
	"fmt"
	"strings"

	"github.com/digitalrebar/store"
	"github.com/gofunky/semver"
)

// All fields must be strings
type ContentMetaData struct {
	// required: true
	Name        string
	Version     string // If present, must be parseable as semver
	Description string
	Source      string // Was who authored it, but was confusing

	// Optional fields
	Documentation    string
	RequiredFeatures string

	// New descriptor fields for catalog
	Color         string
	Icon          string
	Author        string
	DisplayName   string
	License       string
	Copyright     string
	CodeSource    string
	Order         string
	Tags          string // Comma separated list
	DocUrl        string
	Prerequisites string // also a comma-seperated list. May contain semver

	// Informational Fields
	Type         string
	Writable     bool
	Overwritable bool
}

//
// Isos???
// Files??
//
// swagger:model
type Content struct {
	// required: true
	Meta ContentMetaData `json:"meta"`

	/*
		These are the sections:
		tasks        map[string]*models.Task
		bootenvs     map[string]*models.BootEnv
		stages       map[string]*models.Stage
		templates    map[string]*models.Template
		profiles     map[string]*models.Profile
		params       map[string]*models.Param
		reservations map[string]*models.Reservation
		subnets      map[string]*models.Subnet
		users        map[string]*models.User
		preferences  map[string]*models.Pref
		plugins      map[string]*models.Plugin
		machines     map[string]*models.Machine
		leases       map[string]*models.Lease
	*/
	Sections Sections `json:"sections"`
}

func ParseContentPrerequisites(prereqs string) (map[string]semver.Range, error) {
	res := map[string]semver.Range{}
	for _, v := range strings.Split(prereqs, ",") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		parts := strings.SplitN(v, ":", 2)
		if len(parts) == 1 {
			parts = append(parts, ">=0.0.0")
		}
		ver, err := semver.ParseRange(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("Invalid version requirement for %s: %v", parts[0], err)
		}
		res[strings.TrimSpace(parts[0])] = ver
	}
	return res, nil
}

func (c *Content) Prerequisites() (map[string]semver.Range, error) {
	return ParseContentPrerequisites(c.Meta.Prerequisites)
}

func (c *Content) ToStore(dest store.Store) error {
	c.Fill()
	if dmeta, ok := dest.(store.MetaSaver); ok {
		meta := map[string]string{
			"Name":        c.Meta.Name,
			"Version":     c.Meta.Version,
			"Description": c.Meta.Description,
			"Source":      c.Meta.Source,

			"Type": c.Meta.Type,

			"Documentation":    c.Meta.Documentation,
			"RequiredFeatures": c.Meta.RequiredFeatures,

			"Color":         c.Meta.Color,
			"Icon":          c.Meta.Icon,
			"Author":        c.Meta.Author,
			"DisplayName":   c.Meta.DisplayName,
			"License":       c.Meta.License,
			"Copyright":     c.Meta.Copyright,
			"CodeSource":    c.Meta.CodeSource,
			"Order":         c.Meta.Order,
			"Tags":          c.Meta.Tags,
			"DocUrl":        c.Meta.DocUrl,
			"Prerequisites": c.Meta.Prerequisites,
		}
		if err := dmeta.SetMetaData(meta); err != nil {
			return err
		}
	}
	for section, vals := range c.Sections {
		sub, err := dest.MakeSub(section)
		if err != nil {
			return err
		}
		for k, v := range vals {
			if err := sub.Save(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Content) Mangle(thunk func(string, interface{}) (interface{}, error)) error {
	for section := range c.Sections {
		for k := range c.Sections[section] {
			if final, err := thunk(section, c.Sections[section][k]); err == nil && final != nil {
				c.Sections[section][k] = final
			} else if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Content) FromStore(src store.Store) error {
	c.Fill()
	if smeta, ok := src.(store.MetaSaver); ok {
		for k, v := range smeta.MetaData() {
			switch k {
			case "Name":
				c.Meta.Name = v
			case "Source":
				c.Meta.Source = v
			case "Description":
				c.Meta.Description = v
			case "Version":
				c.Meta.Version = v
				if _, err := semver.ParseTolerant(v); v != "" && err != nil {
					return err
				}
			case "Type":
				c.Meta.Type = v
			case "Documentation":
				c.Meta.Documentation = v
			case "RequiredFeatures":
				c.Meta.RequiredFeatures = v
			case "Color":
				c.Meta.Color = v
			case "Icon":
				c.Meta.Icon = v
			case "Author":
				c.Meta.Author = v
			case "DisplayName":
				c.Meta.DisplayName = v
			case "License":
				c.Meta.License = v
			case "Copyright":
				c.Meta.Copyright = v
			case "CodeSource":
				c.Meta.CodeSource = v
			case "Order":
				c.Meta.Order = v
			case "Tags":
				c.Meta.Tags = v
			case "DocUrl":
				c.Meta.DocUrl = v
			case "Prerequisites":
				c.Meta.Prerequisites = v
				if _, err := c.Prerequisites(); err != nil {
					return err
				}
			}
		}
	}
	for section, subStore := range src.Subs() {
		if _, err := New(section); err != nil {
			continue
		}
		keys, err := subStore.Keys()
		if err != nil {
			return err
		}
		c.Sections[section] = map[string]interface{}{}
		for _, key := range keys {
			val, _ := New(section)
			if f, ok := val.(Filler); ok {
				f.Fill()
			}
			if err := subStore.Load(key, val); err != nil {
				return err
			}
			c.Sections[section][key] = val
		}
	}

	c.Meta.Type, c.Meta.Overwritable, c.Meta.Writable = getExtraFields(c.Key(), c.Meta.Type)
	return nil
}

type Sections map[string]Section
type Section map[string]interface{}

func (c *Content) Prefix() string {
	return "contents"
}

func (c *Content) Key() string {
	return c.Meta.Name
}

func (c *Content) KeyName() string {
	return "Meta.Name"
}

func (c *Content) Fill() {
	if c.Sections == nil {
		c.Sections = Sections(map[string]Section{})
	}
}

func (c *Content) AuthKey() string {
	return c.Key()
}

// swagger:model
type ContentSummary struct {
	Meta     ContentMetaData `json:"meta"`
	Counts   map[string]int
	Warnings []string
}

func (c *ContentSummary) Fill() {
	if c.Counts == nil {
		c.Counts = map[string]int{}
	}
	if c.Warnings == nil {
		c.Warnings = []string{}
	}
}

func (c *ContentSummary) FromStore(src store.Store) {
	c.Fill()
	if smeta, ok := src.(store.MetaSaver); ok {
		for k, v := range smeta.MetaData() {
			switch k {
			case "Name":
				c.Meta.Name = v
			case "Source":
				c.Meta.Source = v
			case "Description":
				c.Meta.Description = v
			case "Version":
				c.Meta.Version = v
			case "Type":
				c.Meta.Type = v
			case "Documentation":
				c.Meta.Documentation = v
			case "RequiredFeatures":
				c.Meta.RequiredFeatures = v
			case "Color":
				c.Meta.Color = v
			case "Icon":
				c.Meta.Icon = v
			case "Author":
				c.Meta.Author = v
			case "DisplayName":
				c.Meta.DisplayName = v
			case "License":
				c.Meta.License = v
			case "Copyright":
				c.Meta.Copyright = v
			case "CodeSource":
				c.Meta.CodeSource = v
			case "Order":
				c.Meta.Order = v
			case "DocUrl":
				c.Meta.DocUrl = v
			case "Prerequisites":
				c.Meta.Prerequisites = v
			}
		}
	}
	for section, subStore := range src.Subs() {
		keys, err := subStore.Keys()
		if err != nil {
			continue
		}
		c.Counts[section] = len(keys)
	}

	c.Meta.Type, c.Meta.Overwritable, c.Meta.Writable = getExtraFields(c.Meta.Name, c.Meta.Type)
	return
}

// Return type, overwritable, writable
func getExtraFields(n, t string) (string, bool, bool) {
	writable := false
	overwritable := false
	if t != "" {
		if t == "default" {
			overwritable = true
		}
	} else {
		t = "dynamic"
	}
	if n == "BackingStore" {
		t = "writable"
		writable = true
	} else if n == "LocalStore" {
		t = "local"
		overwritable = true
	} else if n == "BasicStore" {
		t = "basic"
		overwritable = true
	} else if n == "DefaultStore" {
		t = "default"
		overwritable = true
	}
	return t, overwritable, writable
}
