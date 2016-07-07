## An in-memory Go Library for WordNet

This is a go language library for accessing [Princeton's wordnet][].

## Implementation Overview

This library is a native golang parser for wordnet which stores the
entire thing in RAM.  This approach was taken for faster access times
because the wordnet database sits in only about 80MB of ram, which is
not a lot these days.  Parsing the full data files takes around two
seconds on a modest laptop.

[Princeton's wordnet]: http://wordnet.princeton.edu

## Supported features

* Lookup by term
* Synonyms
* All relation types (Antonyms, Hyponyms, Hypernyms, etc)
* Iteration of the database
* Lemmatization

## Missing features

* Morphology - specifically generating a lemma from input text

## Example Usage

```golang
import (
    "log"

    "github.com/lloyd/wnram"
)

func main() {
    wn := wnram.New("./path")
    // lookup "yummy"
    if found, err := wn.Lookup(Criteria{Matching: "yummy", POS: []PartOfSpeech{Adjective}}); err != nil {
       log.Fatal("%s", err)
    } else {
       // dump details about each matching term to console
       for _, f := range found {
           f.Dump()
       }
	}
}
```

## License

BSD 2 Clause, see `LICENSE`.
