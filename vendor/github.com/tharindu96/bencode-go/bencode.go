package bencode

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
)

// BNode can be a string, integer, list or a dictionary
type BNode struct {
	Type BType
	Node interface{}
}

// BString is a string
type BString string

// BInteger is an integer
type BInteger int

// BList is a list of BNodes
type BList []*BNode

// BDictNode is a node in BDict
type BDictNode struct {
	Key   string
	Value *BNode
}

// BDict is a dictionary of BNodes
type BDict []*BDictNode

// BType bencode type
type BType uint

// BencodeType Constants
const (
	BencodeUndefined BType = 0
	BencodeString    BType = 1
	BencodeInteger   BType = 2
	BencodeList      BType = 3
	BencodeDict      BType = 4
)

// BRead read from the input buffer and return a bencode node
func BRead(r *bufio.Reader) (*BNode, error) {
	return parse(r)
}

// GetString returns the string in the node
func (b *BNode) GetString() (BString, error) {
	if b.Type != BencodeString {
		return "", errors.New("not a string")
	}
	s, ok := b.Node.(*BString)
	if !ok {
		return "", errors.New("could not cast")
	}
	return *s, nil
}

// GetInteger returns the integer in the node
func (b *BNode) GetInteger() (BInteger, error) {
	if b.Type != BencodeInteger {
		return 0, errors.New("not an integer")
	}
	i, ok := b.Node.(*BInteger)
	if !ok {
		return 0, errors.New("could not cast")
	}
	return *i, nil
}

// GetList returns the list in the node
func (b *BNode) GetList() (BList, error) {
	if b.Type != BencodeList {
		return nil, errors.New("not a list")
	}
	l, ok := b.Node.(*BList)
	if !ok {
		return nil, errors.New("could not cast")
	}
	return *l, nil
}

// GetDict returns the dict in the node
func (b *BNode) GetDict() (BDict, error) {
	if b.Type != BencodeDict {
		return nil, errors.New("not a dict")
	}
	d, ok := b.Node.(*BDict)
	if !ok {
		return nil, errors.New("could not cast")
	}
	return *d, nil
}

// Print print the bnode to the logger
func (b *BNode) Print() {
	switch b.Type {
	case BencodeString:
		s, ok := b.Node.(*BString)
		if !ok {
			log.Panicln("could not cast node")
		}
		log.Println(*s)
		break
	case BencodeInteger:
		i, ok := b.Node.(*BInteger)
		if !ok {
			log.Panicln("could not cast node")
		}
		log.Println(*i)
		break
	case BencodeList:
		l, ok := b.Node.(*BList)
		if !ok {
			log.Panicln("could not cast node")
		}
		for _, v := range *l {
			v.Print()
		}
		break
	case BencodeDict:
		d, ok := b.Node.(*BDict)
		if !ok {
			log.Panicln("could not case node")
		}
		for _, v := range *d {
			log.Println(v.Key)
			if v.Value.Type == BencodeDict {
				v.Value.Print()
			}
		}
	}
}

// GetBencode returns the bencoded string
func (b *BNode) GetBencode() (string, error) {
	switch b.Type {
	case BencodeString:
		s, err := b.GetString()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d:%s", len(s), s), nil
	case BencodeInteger:
		i, err := b.GetInteger()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("i%de", i), nil
	case BencodeList:
		l, err := b.GetList()
		if err != nil {
			return "", err
		}
		s := "l"
		for _, v := range l {
			x, err := v.GetBencode()
			if err != nil {
				return "", err
			}
			s += x
		}
		s += "e"
		return s, err
	case BencodeDict:
		d, err := b.GetDict()
		if err != nil {
			return "", err
		}
		s := "d"
		for _, v := range d {
			s += fmt.Sprintf("%d:%s", len(v.Key), v.Key)
			x, err := v.Value.GetBencode()
			if err != nil {
				return "", err
			}
			s += x
		}
		s += "e"
		return s, err
	default:
		return "", errors.New("type undefined")
	}
}

// Get returns the node in the dict with the key
func (d *BDict) Get(key string) *BNode {
	for _, v := range *d {
		if v.Key == key {
			return v.Value
		}
	}
	return nil
}

// ToString returns the native string type
func (s *BString) ToString() string {
	if s == nil {
		return ""
	}
	return string(*s)
}

// ToInt returns the native int type
func (i *BInteger) ToInt() int {
	if i == nil {
		return 0
	}
	return int(*i)
}

// BEncode returns BNode of a native go structure
func BEncode(val interface{}) (*BNode, error) {
	s, ok := val.(string)
	if ok {
		bs := BString(s)
		n := &BNode{
			Type: BencodeString,
			Node: &bs,
		}
		return n, nil
	}
	i, ok := val.(int)
	if ok {
		bi := BInteger(i)
		n := &BNode{
			Type: BencodeInteger,
			Node: &bi,
		}
		return n, nil
	}
	l, ok := val.([]interface{})
	if ok {
		bl := make(BList, 0)
		for _, v := range l {
			in, err := BEncode(v)
			if err != nil {
				return nil, err
			}
			bl = append(bl, in)
		}
		n := &BNode{
			Type: BencodeList,
			Node: &bl,
		}
		return n, nil
	}
	d, ok := val.(map[string]interface{})
	if ok {
		bd := make(BDict, 0)
		for k, v := range d {
			in, err := BEncode(v)
			if err != nil {
				return nil, err
			}
			dn := BDictNode{
				Key:   k,
				Value: in,
			}
			bd = append(bd, &dn)
		}
		n := &BNode{
			Type: BencodeDict,
			Node: &bd,
		}
		return n, nil
	}
	return nil, errors.New("unsupported type")
}

func parseString(r *bufio.Reader) (*BString, error) {
	lstr, err := r.ReadString(byte(':'))
	if err != nil {
		return nil, err
	}
	l64, err := strconv.ParseUint(lstr[:len(lstr)-1], 10, 32)
	if err != nil {
		return nil, err
	}
	l := uint(l64)
	buffer := make([]byte, l)
	err = readUntilMax(r, int(l), buffer)
	if err != nil {
		return nil, err
	}
	s := BString(buffer)
	return &s, nil
}

func parseInteger(r *bufio.Reader) (*BInteger, error) {
	istr, err := r.ReadString('e')
	if err != nil {
		return nil, err
	}
	reg, err := regexp.Compile("i-?00+e|i-0e")
	if err != nil {
		return nil, err
	}
	if reg.MatchString(istr) {
		return nil, errors.New("invalid integer")
	}
	i64, err := strconv.ParseInt(istr[1:len(istr)-1], 10, 64)
	i := BInteger(i64)
	return &i, nil
}

func parseList(r *bufio.Reader) (*BList, error) {
	err := discardSafely(r, 1)
	if err != nil {
		return nil, err
	}
	list := make(BList, 0)
	for {
		p, err := r.Peek(1)
		if err != nil {
			return nil, err
		}
		if p[0] == byte('e') {
			err := discardSafely(r, 1)
			if err != nil {
				return nil, err
			}
			return &list, nil
		}
		node, err := parse(r)
		if err != nil {
			return nil, err
		}
		list = append(list, node)
	}
}

func parseDict(r *bufio.Reader) (*BDict, error) {
	err := discardSafely(r, 1)
	if err != nil {
		return nil, err
	}
	dict := make(BDict, 0)
	for {
		p, err := r.Peek(1)
		if err != nil {
			return nil, err
		}
		if p[0] == byte('e') {
			err := discardSafely(r, 1)
			if err != nil {
				return nil, err
			}
			return &dict, nil
		}
		key, err := parseString(r)
		if err != nil {
			return nil, err
		}
		val, err := parse(r)
		if err != nil {
			return nil, err
		}
		n := &BDictNode{
			Key:   key.ToString(),
			Value: val,
		}
		dict = append(dict, n)
	}
}

func parse(r *bufio.Reader) (*BNode, error) {

	ttype, err := getType(r)

	if err != nil {
		return nil, err
	}

	ret := &BNode{}

	switch ttype {
	case BencodeString:
		s, err := parseString(r)
		if err != nil {
			return nil, err
		}
		ret.Type = BencodeString
		ret.Node = s
		break
	case BencodeInteger:
		i, err := parseInteger(r)
		if err != nil {
			return nil, err
		}
		ret.Type = BencodeInteger
		ret.Node = i
		break
	case BencodeList:
		l, err := parseList(r)
		if err != nil {
			return nil, err
		}
		ret.Type = BencodeList
		ret.Node = l
		break
	case BencodeDict:
		d, err := parseDict(r)
		if err != nil {
			return nil, err
		}
		ret.Type = BencodeDict
		ret.Node = d
		break
	default:
		return nil, errors.New("type undefined")
	}

	return ret, nil
}

func discardSafely(r *bufio.Reader, c int) error {
	iskip, err := r.Discard(c)
	if err != nil {
		return err
	}
	if iskip != c {
		return errors.New("discard: invalid number of bytes discarded")
	}
	return nil
}

func readUntilMax(r *bufio.Reader, c int, buffer []byte) error {
	i := 0
	for {
		n, err := r.Read(buffer[i:])
		if err != nil {
			return err
		}
		i += n
		if i == c {
			return nil
		}
	}
}

func getType(r *bufio.Reader) (BType, error) {
	b, err := r.Peek(1)
	if err != nil {
		return BencodeUndefined, err
	}
	switch b[0] {
	case byte('i'):
		return BencodeInteger, nil
	case byte('l'):
		return BencodeList, nil
	case byte('d'):
		return BencodeDict, nil
	default:
		if b[0] >= byte('0') && b[0] <= byte('9') {
			return BencodeString, nil
		}
		return BencodeUndefined, errors.New("type undefined")
	}
}
