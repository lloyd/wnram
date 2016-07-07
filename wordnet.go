package wnram

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// An initialized read-only, in-ram instance of the wordnet database.
// May safely be shared by multiple threads of execution
type Handle struct {
	index map[string][]*cluster
	db    []*cluster
}

type index struct {
	pos   PartOfSpeech
	lemma string
	sense uint8
}

// The results of a search against the wordnet database
type Lookup struct {
	word    string   // the word the user searched for
	cluster *cluster // the discoverd synonym set
}

type syntacticRelation struct {
	rel        Relation
	target     *cluster
	wordNumber uint8
}

type semanticRelation struct {
	rel    Relation
	target *cluster
}

type word struct {
	sense     uint8
	word      string
	relations []syntacticRelation
}

type cluster struct {
	pos       PartOfSpeech
	words     []word
	gloss     string
	relations []semanticRelation
	debug     string
}

// Parts of speech
type PartOfSpeech uint8

// A set of multiple parts of speech
type PartOfSpeechList []PartOfSpeech

func (l PartOfSpeechList) Empty() bool {
	return len(l) == 0
}

func (l PartOfSpeechList) Contains(want PartOfSpeech) bool {
	for _, got := range l {
		if got == want {
			return true
		}
	}
	return false
}

const (
	Noun PartOfSpeech = iota
	Verb
	Adjective
	//	AdjectiveSatellite
	Adverb
)

func (pos PartOfSpeech) String() string {
	switch pos {
	case Noun:
		return "noun"
	case Verb:
		return "verb"
	case Adjective:
		return "adj"
	case Adverb:
		return "adv"
	}
	return "unknown"
}

// The ways in which synonym clusters may be related to others.
type Relation uint32

const (
	AlsoSee Relation = 1 << iota
	// A word with an opposite meaning
	Antonym
	// A noun for which adjectives express values.
	// The noun weight is an attribute, for which the adjectives light and heavy express values.
	Attribute
	Cause
	// Terms in different syntactic categories that have the same root form and are semantically related.
	DerivationallyRelatedForm
	// Adverbs are often derived from adjectives, and sometimes have antonyms; therefore the synset for an adverb usually contains a lexical pointer to the adjective from which it is derived.
	DerivedFromAdjective
	// A topical classification to which a synset has been linked with a REGION
	InDomainRegion
	InDomainTopic
	InDomainUsage
	ContainsDomainRegion
	ContainsDomainTopic
	ContainsDomainUsage
	Entailment
	// The generic term used to designate a whole class of specific instances.
	// Y is a hypernym of X if X is a (kind of) Y .
	Hypernym
	InstanceHypernym
	InstanceHyponym
	// The specific term used to designate a member of a class. X is a hyponym of Y if X is a (kind of) Y .
	Hyponym
	MemberMeronym
	PartMeronym
	SubstanceMeronym
	MemberHolonym
	PartHolonym
	SubstanceHolonym
	ParticipleOfVerb
	RelatedForm
	SimilarTo
	VerbGroup
)
const Pertainym = DerivedFromAdjective

func (w *Lookup) String() string {
	return fmt.Sprintf("%q (%s)", w.word, w.cluster.pos.String())
}

// The specific word that was found
func (w *Lookup) Word() string {
	return w.word
}

// A canonical synonym for this word
func (w *Lookup) Lemma() string {
	return w.cluster.words[0].word
}

func (w *Lookup) DumpStr() string {
	s := fmt.Sprintf("Word: %s\n", w.String())
	s += fmt.Sprintf("Synonyms: ")
	words := []string{}
	for _, w := range w.cluster.words {
		words = append(words, w.word)
	}
	s += strings.Join(words, ", ") + "\n"
	s += fmt.Sprintf("%d semantic relationships\n", len(w.cluster.relations))
	s += "| " + w.cluster.gloss + "\n"
	return s
}

func (w *Lookup) Dump() {
	fmt.Printf("%s", w.DumpStr())
}

func (w *Lookup) POS() PartOfSpeech {
	return w.cluster.pos
}

func (w *Lookup) Synonyms() (synonyms []string) {
	for _, w := range w.cluster.words {
		synonyms = append(synonyms, w.word)
	}
	return synonyms
}

// Get words related to this word.  r is a bitfield of relation types
// to include
func (w *Lookup) Related(r Relation) (relationships []Lookup) {
	// first look for semantic relationships
	for _, rel := range w.cluster.relations {
		if rel.rel&r != Relation(0) {
			relationships = append(relationships, Lookup{
				word:    rel.target.words[0].word,
				cluster: rel.target,
			})
		}
	}
	// next let's look for syntactic relationships
	key := normalize(w.word)
	for _, word := range w.cluster.words {
		if key == word.word {
			for _, rel := range word.relations {
				if rel.rel&r != Relation(0) {
					relationships = append(relationships, Lookup{
						word:    rel.target.words[rel.wordNumber].word,
						cluster: rel.target,
					})
				}
			}
		}
	}

	return relationships
}

// Initialize a new in-ram WordNet databases reading files from the
// specified directory.
func New(dir string) (*Handle, error) {
	cnt := 0
	type ix struct {
		index string
		pos   PartOfSpeech
	}
	byOffset := map[ix]*cluster{}
	err := filepath.Walk(dir, func(filename string, info os.FileInfo, err error) error {
		start := time.Now()
		if err != nil || info.IsDir() {
			return err
		}

		// Skip '^.', '~$', and non-files.
		if strings.HasPrefix(path.Base(filename), ".") || strings.HasSuffix(filename, "~") || strings.HasSuffix(filename, "#") {
			return nil
		}
		// read only data files
		if !strings.HasPrefix(path.Base(filename), "data") {
			return nil
		}

		err = inPlaceReadLineFromPath(filename, func(data []byte, line, offset int64) error {
			cnt++
			if p, err := parseLine(data, line, offset); err != nil {
				return fmt.Errorf("%s:%d: %s", err)
			} else if p != nil {
				// first, let's identify the cluster
				index := ix{p.byteOffset, p.pos}
				c, ok := byOffset[index]
				if !ok {
					c = &cluster{}
					byOffset[index] = c
				}
				// now update
				c.pos = p.pos
				c.words = p.words
				c.gloss = p.gloss
				c.debug = p.byteOffset

				// now let's build relations
				for _, r := range p.rels {
					rindex := ix{r.offset, r.pos}
					rcluster, ok := byOffset[rindex]
					if !ok {
						// create the other side of the relationship
						rcluster = &cluster{}
						byOffset[rindex] = rcluster
					}
					if r.isSemantic {
						c.relations = append(c.relations, semanticRelation{
							rel:    r.rel,
							target: rcluster,
						})
					} else {
						if int(r.source) >= len(c.words) {
							return fmt.Errorf("%s:%d: error parsing relations, bogus source (words: %d, offset: %d) [%s]", filename, line, r.source, len(c.words), string(data))
						}
						c.words[r.source].relations = append(c.words[r.source].relations, syntacticRelation{
							rel:        r.rel,
							target:     rcluster,
							wordNumber: r.dest,
						})
					}
				}

			}
			return nil
		})
		fmt.Printf("%s in %s\n", filename, time.Since(start).String())
		return err
	})
	if err != nil {
		return nil, err
	}

	// now that we've built up the in ram database, lets' index it
	h := Handle{
		db:    make([]*cluster, 0, len(byOffset)),
		index: make(map[string][]*cluster),
	}
	for _, c := range byOffset {
		if len(c.words) == 0 {
			return nil, fmt.Errorf("ERROR, internal consistency error -> cluster without words %v\n", c)
		}
		// add to the global slice of synsets (supports iteration)
		h.db = append(h.db, c)

		// now index all the strings
		for _, w := range c.words {
			key := normalize(w.word)
			v, _ := h.index[key]
			v = append(v, c)
			h.index[key] = v
		}
	}

	return &h, nil
}

type Criteria struct {
	Matching string
	POS      PartOfSpeechList
}

func normalize(in string) string {
	return strings.ToLower(strings.Join(strings.Fields(in), " "))
}

// look up word clusters based on given criteria
func (h *Handle) Lookup(crit Criteria) ([]Lookup, error) {
	if crit.Matching == "" {
		return nil, fmt.Errorf("empty string passed as criteria to lookup")
	}
	searchStr := normalize(crit.Matching)
	clusters, _ := h.index[searchStr]
	found := []Lookup{}
	for _, c := range clusters {
		if len(crit.POS) > 0 {
			satisfied := false
			for _, p := range crit.POS {
				if p == c.pos {
					satisfied = true
					break
				}
			}
			if !satisfied {
				continue
			}
		}
		found = append(found, Lookup{
			word:    crit.Matching,
			cluster: c,
		})
	}
	return found, nil
}

func (h *Handle) Iterate(pos PartOfSpeechList, cb func(Lookup) error) error {
	for _, c := range h.db {
		if !pos.Empty() && !pos.Contains(c.pos) {
			continue
		}
		err := cb(Lookup{
			word:    c.words[0].word,
			cluster: c,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
