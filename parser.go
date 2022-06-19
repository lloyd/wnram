package wnram

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type tokenType uint8

const (
	tNone tokenType = iota
	tEOL
	tNumber
	tPOS
)

func (t tokenType) GoString() string {
	return t.String()
}

func (t tokenType) String() string {
	switch t {
	case tNone:
		return "none"
	case tEOL:
		return "eol"
	case tNumber:
		return "number"
	case tPOS:
		return "pos"
	}
	return "unknown"
}

type token struct {
	t tokenType
	v string
	n int64
	p PartOfSpeech
}

type lexable string

func (l *lexable) chomp() {
	for {
		if c, ok := l.peek(); ok && unicode.IsSpace(c) {
			l.next()
		} else {
			break
		}
	}
}

func (l *lexable) next() (rune, bool) {
	if !l.empty() {
		curchar, width := utf8.DecodeRuneInString(string(*l))
		*l = (*l)[width:]
		return curchar, true
	}
	return ' ', false
}
func (l *lexable) peek() (rune, bool) {
	if !l.empty() {
		c, _ := utf8.DecodeRuneInString(string(*l))
		return c, true
	}
	return ' ', false
}

func (l *lexable) empty() bool {
	return len(*l) == 0
}

func (l *lexable) lexDecimalNumber() (int64, error) {
	l.chomp()
	var number string
	for {
		if r, ok := l.peek(); !ok || !unicode.IsDigit(r) {
			break
		}
		r, _ := l.next()
		number += fmt.Sprintf("%c", r)
	}
	if len(number) == 0 {
		return 0, fmt.Errorf("number not found in string: %s", *l)
	}
	i, err := strconv.ParseInt(number, 10, 64)
	if err != nil {
		return 0, err
	}
	return i, nil
}

func (l *lexable) lexWord() (string, error) {
	l.chomp()
	var word []byte
	buf := make([]byte, 4, 4)
	for {
		if r, ok := l.peek(); !ok || unicode.IsSpace(r) {
			break
		}
		r, _ := l.next()
		if r == '_' {
			word = append(word, byte(' '))
		} else {
			x := utf8.EncodeRune(buf, r)
			word = append(word, buf[0:x]...)
		}
	}
	return string(word), nil
}

func (l *lexable) lexGloss() (string, error) {
	l.chomp()
	r, ok := l.next()
	if !ok {
		return "", fmt.Errorf("definition expected")
	}
	if r != '|' {
		return "", fmt.Errorf("definition expected (want '|' got '%c') [%q]", r, string(*l))
	}
	return strings.TrimSpace(string(*l)), nil
}

func (l *lexable) lexHexNumber() (int64, error) {
	l.chomp()
	var number string
	for {
		if r, ok := l.peek(); !ok || !unicode.Is(unicode.Hex_Digit, r) {
			break
		}
		r, _ := l.next()
		number += fmt.Sprintf("%c", r)
	}
	if len(number) == 0 {
		return 0, fmt.Errorf("number not found in string: %s", *l)
	}
	i, err := strconv.ParseInt(number, 16, 64)
	if err != nil {
		return 0, err
	}
	return i, nil
}

func (l *lexable) lexOffset() (string, error) {
	l.chomp()
	if len(*l) < 8 {
		return "", fmt.Errorf("invalid offset")
	}
	for i := 0; i < 8; i++ {
		if !unicode.IsDigit(rune((*l)[i])) {
			return "", fmt.Errorf("invalid chars in offset: %s", string((*l)[0:8]))
		}
	}
	cpy := make([]byte, 8, 8)
	copy(cpy, (*l)[0:8])
	*l = (*l)[8:]
	return string(cpy), nil
}

func (l *lexable) lexPOS() (PartOfSpeech, error) {
	l.chomp()
	curchar, ok := l.next()
	if !ok {
		return 0, fmt.Errorf("unexpected end of input")
	}
	switch curchar {
	case 'n':
		return Noun, nil
	case 'v':
		return Verb, nil
	case 'a':
		return Adjective, nil
	case 's':
		// XXX: note, that an adjective is not the core of an adj cluster is not
		// really related to the part of speech.  its more like encoding a relationship
		// between the adjective and the head cluster.  For now, let's just smash it into
		// adjective.
		return Adjective, nil
	case 'r':
		return Adverb, nil
	}
	return 0, fmt.Errorf("invalid part of speech: %c", curchar)
}

func (l *lexable) lexRelationType() (Relation, error) {
	l.chomp()
	word, err := l.lexWord()
	if err != nil {
		return 0, fmt.Errorf("can't read relation type: %s", err)
	}
	switch word {
	case "!":
		return Antonym, nil
	case "#m":
		return MemberHolonym, nil
	case "#p":
		return PartHolonym, nil
	case "#s":
		return SubstanceHolonym, nil
	case "$":
		return VerbGroup, nil
	case "%m":
		return MemberMeronym, nil
	case "%p":
		return PartMeronym, nil
	case "%s":
		return SubstanceMeronym, nil
	case "&":
		return SimilarTo, nil
	case "*":
		return Entailment, nil
	case "+":
		return DerivationallyRelatedForm, nil
	case "-c":
		return InDomainTopic, nil
	case "-r":
		return InDomainRegion, nil
	case "-u":
		return InDomainUsage, nil
	case ";c":
		return ContainsDomainTopic, nil
	case ";r":
		return ContainsDomainRegion, nil
	case ";u":
		return ContainsDomainUsage, nil
	case "<":
		return ParticipleOfVerb, nil
	case "=":
		return Attribute, nil
	case ">":
		return Cause, nil
	case "@":
		return Hypernym, nil
	case "@i":
		return InstanceHypernym, nil
	case "\\":
		// also DerivedFromAdjective
		return Pertainym, nil
	case "^":
		return AlsoSee, nil
	case "~":
		return Hyponym, nil
	case "~i":
		return InstanceHyponym, nil
	}
	return 0, fmt.Errorf("unrecognized pointer type: %q", word)
}

type parsedRel struct {
	pos          PartOfSpeech
	rel          Relation
	offset       string
	isSemantic   bool
	source, dest uint8
}

type parsed struct {
	byteOffset string
	pos        PartOfSpeech
	fileNum    int64
	words      []word
	gloss      string
	rels       []parsedRel
}

func parseLine(data []byte, line, offset int64) (*parsed, error) {
	l := lexable(data)

	l.chomp()
	byteOffset, err := l.lexOffset()
	if err != nil {
		// was this a comment line?
		n, err := l.lexDecimalNumber()
		if err == nil && n == line {
			// comment!
			return nil, nil
		}
		return nil, fmt.Errorf("can't parse line, expected comment or Offset")
	}
	// file number
	filenum, err := l.lexDecimalNumber()
	if err != nil {
		return nil, fmt.Errorf("filenumber expected: %s", err)
	}
	pos, err := l.lexPOS()
	if err != nil {
		return nil, fmt.Errorf("part of speech expected: %s", err)
	}
	// lexicographer file containing the word
	wordcount, err := l.lexHexNumber()
	if err != nil {
		return nil, fmt.Errorf("wordcount expected: %s", err)
	}
	p := parsed{
		byteOffset: byteOffset,
		pos:        pos,
		fileNum:    filenum,
	}
	for ; wordcount > 0; wordcount-- {
		value, err := l.lexWord()
		if err != nil {
			return nil, fmt.Errorf("word expected on line %d", line)
		}
		// XXX: handle syntactic markers
		sense, err := l.lexHexNumber()
		if err != nil {
			return nil, fmt.Errorf("sense id expected on line %d", line)
		}
		p.words = append(p.words, word{
			word:  value,
			sense: uint8(sense),
		})
	}
	pcount, err := l.lexDecimalNumber()
	if err != nil {
		return nil, fmt.Errorf("pointer count expected: %s", err)
	}
	for ; pcount > 0; pcount-- {
		if rt, err := l.lexRelationType(); err != nil {
			return nil, err
		} else if offset, err := l.lexOffset(); err != nil {
			return nil, err
		} else if pos, err := l.lexPOS(); err != nil {
			return nil, err
		} else if nature, err := l.lexHexNumber(); err != nil {
			return nil, err
		} else {
			r := parsedRel{
				rel:    rt,
				pos:    pos,
				offset: offset,
			}
			if nature == 0 {
				r.isSemantic = true
			} else {
				r.isSemantic = false
				r.source = uint8(nature>>8) - 1
				r.dest = uint8(nature&0xff) - 1
			}
			p.rels = append(p.rels, r)
		}
	}
	// parse optional frame count
	frameCount, err := l.lexDecimalNumber()
	if err == nil {
		for ; frameCount > 0; frameCount-- {
			l.chomp()
			if r, ok := l.next(); !ok || r != '+' {
				return nil, fmt.Errorf("missing frame marker (+)")
			} else if _, err := l.lexDecimalNumber(); err != nil {
				return nil, fmt.Errorf("malformed frame number: %s", err)
			} else if _, err := l.lexHexNumber(); err != nil {
				return nil, fmt.Errorf("malformed word number in frame: %s", err)
			}
		}
	}
	gloss, err := l.lexGloss()
	if err != nil {
		return nil, err
	}
	p.gloss = gloss

	return &p, nil
}
