# A lorem ipsum generator for Go

[![Build Status](https://travis-ci.org/go-loremipsum/loremipsum.svg?branch=master)](https://travis-ci.org/go-loremipsum/loremipsum)
[![Coverage Status](https://coveralls.io/repos/github/go-loremipsum/loremipsum/badge.svg?branch=master)](https://coveralls.io/github/go-loremipsum/loremipsum?branch=master)
[![Go Report Card](https://goreportcard.com/badge/gopkg.in/loremipsum.v1)](https://goreportcard.com/report/gopkg.in/loremipsum.v1)
[![GoDoc](https://godoc.org/gopkg.in/loremipsum.v1?status.svg)](https://godoc.org/gopkg.in/loremipsum.v1)

## Usage

~~~go
import "gopkg.in/loremipsum.v1"

loremIpsumGenerator := loremipsum.New()
~~~

Or, if you want repeatable output, pass seed

~~~go
import "gopkg.in/loremipsum.v1"

loremIpsumGenerator := loremipsum.NewWithSeed(1234)
~~~


### Generate random lorem ipsum word

~~~go
word := loremIpsumGenerator.Word()
fmt.Println(word)
~~~

~~~
interdum
~~~

### Generate random lorem ipsum words

~~~go
words := loremIpsumGenerator.Words(5)
fmt.Println(words)
~~~

~~~
lorem ipsum dolor sit amet
~~~

### Generate random lorem ipsum sentence

~~~go
sentence := loremIpsumGenerator.Sentence()
fmt.Println(sentence)
~~~

~~~
Lorem ipsum dolor sit amet consectetur adipiscing elit tincidunt inceptos quis faucibus nunc maecenas nostra eros nam mollis augue habitasse mattis per enim odio suscipit.
~~~

### Generate random lorem ipsum sentences

~~~go
sentences := loremIpsumGenerator.Sentences(2)
fmt.Println(sentences)
~~~

~~~
Lorem ipsum dolor sit amet consectetur adipiscing elit id sem facilisi sapien nibh curabitur nam viverra lacinia. Luctus conubia pulvinar ornare natoque hendrerit dui praesent libero porttitor at suspendisse amet viverra nisl tristique hac ad eget semper et ligula vulputate.
~~~

### Generate random lorem ipsum paragraph

~~~go
paragraph := loremIpsumGenerator.Paragraph()
fmt.Println(paragraph)
~~~

~~~
Lorem ipsum dolor sit amet consectetur adipiscing elit cubilia lobortis efficitur nunc diam egestas cursus laoreet interdum integer rutrum lacus elementum venenatis. Tincidunt feugiat nam hendrerit bibendum suspendisse interdum rhoncus, diam inceptos auctor nisi non vivamus dictum platea tristique euismod egestas odio lacinia. Fusce placerat dolor dui massa venenatis bibendum auctor magnis nisl molestie euismod ipsum rutrum neque lorem nisi justo odio a. Lectus pretium sem a mi eleifend interdum mus porta rutrum suspendisse quis cubilia habitasse luctus dolor metus aenean. Imperdiet amet magna etiam vulputate sollicitudin facilisis mollis duis taciti tristique magnis fusce porttitor. At scelerisque sapien amet venenatis a finibus neque quam dictum natoque tempus ridiculus porttitor ultricies diam luctus dis enim.
~~~

### Generate random lorem ipsum paragraphs

~~~go
paragraphs := loremIpsumGenerator.Paragraphs(3)
fmt.Println(paragraphs)
~~~

~~~
Lorem ipsum dolor sit amet consectetur adipiscing elit habitasse nascetur arcu orci nisl torquent rutrum aenean nisi primis felis fusce dui. Suscipit hac montes neque duis dignissim sem pharetra sit laoreet eu curabitur vivamus class eleifend. Sagittis porta suspendisse felis turpis vehicula ad habitant dignissim pulvinar himenaeos at consectetur morbi luctus faucibus ultricies euismod volutpat maecenas. Sollicitudin facilisis ligula platea litora maecenas molestie rhoncus fermentum velit porta eu dictumst laoreet donec class potenti etiam bibendum sagittis inceptos dapibus magna pharetra porttitor suscipit aptent convallis nulla lacinia eget.
Fames libero id nunc eu malesuada nisl feugiat quam purus enim quisque porttitor velit dolor augue etiam tempor dictumst neque mattis conubia facilisis ullamcorper scelerisque natoque lacus fusce proin pharetra magnis rhoncus. Ante condimentum mi vel odio class nullam nostra nam taciti vitae nec potenti maecenas sit sodales ligula tincidunt montes pretium eu. Ornare euismod mollis ex augue lacus aliquam habitant mi donec sollicitudin consequat rutrum finibus fames lobortis bibendum leo ullamcorper gravida ac turpis ultricies convallis dictumst erat amet. Nostra lorem semper nisi fringilla ac integer odio dolor fusce sociosqu sollicitudin habitasse lacinia mauris blandit montes imperdiet nunc urna. Aliquam semper rutrum amet nam cursus donec turpis ut interdum convallis felis finibus luctus risus posuere.
Enim massa hendrerit fames faucibus tempor porta mi laoreet habitasse ligula purus rutrum facilisis interdum donec varius fringilla nibh nam eleifend. Lacinia tempor augue quis ut tortor eleifend varius facilisis sagittis pharetra, feugiat habitant magna porttitor iaculis nisl pellentesque egestas maximus praesent habitasse congue nostra elementum luctus nam potenti euismod etiam torquent class. Hendrerit euismod cras egestas tempus congue parturient, ultrices finibus dictumst pharetra eleifend donec elementum sollicitudin lobortis magna nascetur dolor curabitur. Eros ac feugiat ridiculus fringilla fusce adipiscing, massa ad est ornare vitae facilisis parturient molestie leo mauris rutrum lectus a.
~~~
