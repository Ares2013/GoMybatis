package GoMybatis

import (
	"bytes"
	"github.com/zhuxiujia/GoMybatis/lib/github.com/beevik/etree"
	"github.com/zhuxiujia/GoMybatis/utils"
	"reflect"
	"strings"
)

const EtreeCharData = `*etree.CharData`
const EtreeElement = `*etree.Element`

const Element_Mapper = "mapper"
const ID = `id`

type MapperXml struct {
	Tag          string
	Id           string
	Propertys    map[string]string
	ElementItems []ElementItem
}

type ElementItem struct {
	ElementType  string
	Propertys    map[string]string
	DataString   string
	ElementItems []ElementItem
}

//load xml from string data,return a map[elementId]*MapperXml
func LoadMapperXml(bytes []byte) (items map[string]*MapperXml) {
	utils.FixTestExpressionSymbol(&bytes)
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(bytes); err != nil {
		panic(err)
	}
	items = make(map[string]*MapperXml)
	root := doc.SelectElement(Element_Mapper)
	for _, s := range root.ChildElements() {
		var attrMap = attrToProperty(s.Attr)
		var elItems = loop(s)
		if s.Tag == Element_Insert ||
			s.Tag == Element_Delete ||
			s.Tag == Element_Update ||
			s.Tag == Element_Select ||
			s.Tag == Element_ResultMap ||
			s.Tag == Element_Sql ||
			s.Tag == Element_Insert_Templete ||
			s.Tag == Element_Delete_Templete ||
			s.Tag == Element_Update_Templete ||
			s.Tag == Element_Select_Templete {
			var elementID = attrMap[ID]

			if elementID == "" {
				//如果id不存在，id设置为tag
				attrMap[ID] = s.Tag
				elementID = s.Tag
			}
			if elementID != "" {
				var oldItem = items[elementID]
				if oldItem != nil {
					panic("[GoMybatis] element Id can not repeat in xml! elementId=" + elementID)
				}
			}
			var mapperXml = MapperXml{
				Tag:          s.Tag,
				Id:           elementID,
				ElementItems: elItems,
				Propertys:    attrMap,
			}
			items[elementID] = &mapperXml
		}
	}
	for itemsIndex, mapperXml := range items {
		for key, v := range mapperXml.ElementItems {
			var isChanged = includeElementReplace(&v, &items)
			if isChanged {
				mapperXml.ElementItems[key] = v
			}
		}
		items[itemsIndex] = mapperXml
	}
	return items
}

func includeElementReplace(xml *ElementItem, xmlMap *map[string]*MapperXml) bool {
	var changed = false
	if xml.ElementType == Element_Include {
		var refid = xml.Propertys["refid"]
		if refid == "" {
			panic(`[GoMybatis] xml <includ refid=""> 'refid' can not be ""`)
		}
		var mapperXml = (*xmlMap)[refid]
		if mapperXml == nil {
			panic(`[GoMybatis] xml <includ refid="` + refid + `"> element can not find !`)
		}
		if xml != nil {
			(*xml).ElementItems = mapperXml.ElementItems
			changed = true
		}
	}
	if xml.ElementItems != nil {
		for index, v := range xml.ElementItems {
			var isChanged = includeElementReplace(&v, xmlMap)
			if isChanged {
				xml.ElementItems[index] = v
			}
		}
	}
	return changed
}

func attrToProperty(attrs []etree.Attr) map[string]string {
	var m = make(map[string]string)
	for _, v := range attrs {
		m[v.Key] = v.Value
	}
	return m
}

func loop(fatherElement *etree.Element) []ElementItem {
	var els = make([]ElementItem, 0)
	for _, el := range fatherElement.Child {
		var typeString = reflect.ValueOf(el).Type().String()
		if typeString == EtreeCharData {
			var d = el.(*etree.CharData)
			var str = d.Data
			if str == "" {
				continue
			}
			str = strings.Replace(str, "\n", "", -1)
			str = strings.Replace(str, "\t", "", -1)
			str = strings.Trim(str, " ")
			if str != "" {
				var buf bytes.Buffer
				buf.WriteString(" ")
				buf.WriteString(str)
				var elementItem = ElementItem{
					ElementType: Element_String,
					DataString:  buf.String(),
				}
				els = append(els, elementItem)
			}
		} else if typeString == EtreeElement {
			var e = el.(*etree.Element)
			var element = ElementItem{
				ElementType:  e.Tag,
				ElementItems: make([]ElementItem, 0),
				Propertys:    attrToProperty(e.Attr),
			}
			elementRuleCheck(fatherElement, element)
			if len(e.Child) > 0 {
				var loopEls = loop(e)
				for _, item := range loopEls {
					element.ElementItems = append(element.ElementItems, item)
				}
			}
			els = append(els, element)
		}
	}
	return els
}

//标签上下级关系检查
func elementRuleCheck(fatherElement *etree.Element, childElementItem ElementItem) {
	if fatherElement.Tag != Element_choose && (childElementItem.ElementType == Element_when || childElementItem.ElementType == Element_otherwise) {
		panic("[GoMybatis] find element <" + childElementItem.ElementType + "> not in <choose>!")
	}
}
