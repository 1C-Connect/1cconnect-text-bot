package us

import (
	"encoding/xml"
	"errors"
	"fmt"
)

type (
	PropertyValue struct {
		Type string `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
		Text string `xml:",chardata"`
	}

	ParamsProperty struct {
		XMLName xml.Name      `xml:"http://v8.1c.ru/8.1/data/core Property"`
		Name    string        `xml:"name,attr"`
		Value   PropertyValue `xml:"Value"`
	}

	Params struct {
		Property []ParamsProperty `xml:"Property,omitempty"`
	}
)

type (
	PropertyValueTableRowValue struct {
		Type string `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
		Text string `xml:",chardata"`
	}

	PropertyValueTableColumnValueType struct {
		Type string `xml:"Type"`
	}

	PropertyValueTableColumn struct {
		Name string `xml:"Name"`

		ValueType PropertyValueTableColumnValueType `xml:"ValueType"`
	}

	PropertyValueTableRow struct {
		Value []PropertyValueTableRowValue `xml:"Value"`
	}

	PropertyValueTable struct {
		Type string `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`

		Text string `xml:",chardata"`

		Column []PropertyValueTableColumn `xml:"column,omitempty"`
		Row    []PropertyValueTableRow    `xml:"row,omitempty"`
	}

	ParamsPropertyTable struct {
		XMLName xml.Name           `xml:"http://v8.1c.ru/8.1/data/core Property"`
		Name    string             `xml:"name,attr"`
		Value   PropertyValueTable `xml:"Value"`
	}

	ParamsTable struct {
		Property []ParamsPropertyTable `xml:"Property,omitempty"`
	}
)

type (
	PropertyValuePropertyValueStructure struct {
		Text string `xml:",chardata"`
		Type string `xml:"type,attr"`
	}

	PropertyValuePropertyStructure struct {
		Type string `xml:"type,attr,omitempty"`

		Name string `xml:"name,attr,omitempty"`

		Value PropertyValuePropertyValueStructure `xml:"Value"`
	}

	PropertyValueStructure struct {
		Type string `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`

		Text string `xml:",chardata"`

		Property []PropertyValuePropertyStructure `xml:"Property,omitempty"`
	}

	ParamsPropertyStructure struct {
		XMLName xml.Name               `xml:"http://v8.1c.ru/8.1/data/core Property"`
		Name    string                 `xml:"name,attr"`
		Value   PropertyValueStructure `xml:"Value"`
	}

	ParamsStructure struct {
		Property []ParamsPropertyStructure `xml:"Property,omitempty"`
	}
)

func (p *ParamsTable) GetResult() (result PropertyValueTable, err error) {
	for _, v := range p.Property {
		switch v.Name {
		case ResultCode:
			if v.Value.Text != SUCCESS {
				err = errors.New(fmt.Sprint("ResultCode:", v.Value.Text))
			}
		case ResultData:
			if err != nil {
				err = errors.New(fmt.Sprint(err, " ResultData:", v.Value.Text))
			}
			result = v.Value
		}
	}

	return
}

func (p *ParamsStructure) GetResult() (result []PropertyValuePropertyStructure, err error) {
	for _, v := range p.Property {
		switch v.Name {
		case ResultCode:
			if v.Value.Text != SUCCESS {
				err = errors.New(fmt.Sprint("ResultCode:", v.Value.Text))
			}
		case ResultData:
			if err != nil {
				err = errors.New(fmt.Sprint(err, " ResultData:", v.Value.Text))
			}
			result = v.Value.Property
		}
	}

	return
}

func (p *Params) GetResult() (result PropertyValue, err error) {
	for _, v := range p.Property {
		switch v.Name {
		case ResultCode:
			if v.Value.Text != SUCCESS {
				err = errors.New(fmt.Sprint("ResultCode:", v.Value.Text))
			}
		case ResultData:
			if err != nil {
				err = errors.New(fmt.Sprint(err, " ResultData:", v.Value.Text))
			}
			result = v.Value
		}
	}

	return
}
