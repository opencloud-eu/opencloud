# Yet Another CLi Spinner (for Go)
[![License](https://img.shields.io/github/license/theckman/yacspin.svg)](https://github.com/theckman/yacspin/blob/master/LICENSE)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/theckman/yacspin)
[![Latest Git Tag](https://img.shields.io/github/tag/theckman/yacspin.svg)](https://github.com/theckman/yacspin/releases)
[![GitHub Actions master Build Status](https://github.com/theckman/yacspin/actions/workflows/tests.yaml/badge.svg?branch=master)](https://github.com/theckman/yacspin/actions/workflows/tests.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/theckman/yacspin)](https://goreportcard.com/report/github.com/theckman/yacspin)
[![Codecov](https://img.shields.io/codecov/c/github/theckman/yacspin)](https://codecov.io/gh/theckman/yacspin)

Package `yacspin` provides yet another CLi spinner for Go, taking inspiration
(and some utility code) from the https://github.com/briandowns/spinner project.
Specifically `yacspin` borrows the default character sets, and color mappings to
github.com/fatih/color colors, from that project.

## License
Because this package adopts the spinner character sets from https://github.com/briandowns/spinner,
this package is released under the Apache 2.0 License.

## Yet Another CLi Spinner?
This project was created after it was realized that the most popular spinner
library for Go had some limitations, that couldn't be fixed without a massive
overhaul of the API.

The other spinner ties the ability to show updated messages to the spinner's
animation, meaning you can't always show all the information you want to the end
user without changing the animation speed. This means you need to trade off
animation aesthetics to show "realtime" information. It was a goal to avoid this
problem.

In addition, there were also some API design choices that have made it unsafe
for concurrent use, which presents challenges when trying to update the text in
the spinner while it's animating. This could result in undefined behavior due to
data races.

There were also some variable-width spinners in that other project that did
not render correctly. Because the width of the spinner animation would change,
so would the position of the message on the screen. `yacspin` uses a dynamic
width when animating, so your message should appear static relative to the
animating spinner.

Finally, there was an interest in the spinner being able to represent a task, and to
indicate whether it failed or was successful. This would have further compounded
the API changes needed above to support in an intuitive way.

This project takes inspiration from that other project, and takes a new approach
to address the challenges above.

## Features
#### Provided Spinners
There are over 90 spinners available in the `CharSets` package variable. They
were borrowed from [github.com/briandowns/spinner](https://github.com/briandowns/spinner).
There is a table with most of the spinners [at the bottom of this README](#Spinners).

#### Dynamic Width of Animation
Because of how some spinners are animated, they may have different widths are
different times in the animation. `yacspin` calculates the maximum width, and
pads the animation to ensure the text's position on the screen doesn't change.
This results in a smoother looking animation.

##### yacspin
![yacspin animation with dynamic width](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/features/width_good.gif)

##### other spinners
![other spinners' animation with dynamic width](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/features/width_bad.gif)

#### Success and Failure Results
The spinner has both a `Stop()` and `StopFail()` method, which allows the
spinner to result in a success message or a failure message. The messages,
colors, and even the character used to denote success or failure are
customizable in either the initial config or via the spinner's methods.

By doing this you can use a single `yacspin` spinner to display the status of a
list of tasks being executed serially.

##### Stop
![Animation with Success](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/features/stop.gif)

##### StopFail
![Animation with Failure](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/features/stop_fail.gif)

#### Animation At End of Line
The `SpinnerAtEnd` field of the `Config` struct allows you to specify whether
the spinner is rendered at the end of the line instead of the beginning. The
default value (`false`) results in the spinner being rendered at the beginning
of the line.

#### Concurrency
The spinner is safe for concurrent use, so you can update any of its settings
via methods whether the spinner is stopped or is currently animating.

#### Live Updates
Most spinners tie the ability to show new messages with the animation of the
spinner. So if the spinner animates every 200ms, you can only show updated
information every 200ms. If you wanted more frequent updates, you'd need to
tradeoff the asthetics of the animation to display more data.

This spinner updates the printed information of the spinner immediately on
change, without the animation updating. This allows you to use an animation
speed that looks astheticaly pleasing, while also knowing the data presented to
the user will be updated live.

You can see this in action in the following gif, where the filenames being
uploaded are rendered independent of the spinner being animated:

![Animation with Success](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/features/stop.gif)

#### Pausing for Updates
Sometimes you want to change a few settings, and don't want the spinner to
render your partially applied configuration. If your spinner is running, and you
want to change a few configuration items via method calls, you can `Pause()` the
spinner first. After making the changes you can call `Unpause()`, and it will
continue rendering like normal with the newly applied configuration.

#### Supporting Non-Interactive (TTY) Output Targets
`yacspin` also has native support for non-interactive (TTY) output targets. By
default this is detected in the constructor, or can be overriden via the
`TerminalMode` `Config` struct field. When detecting the application is not
running withn a TTY session, the behavior of the spinner is different.

Specifically, when this is automatically detected the spinner no longer uses
colors, disables the automatic spinner animation, and instead only animates the
spinner when updating the message. In addition, each animation is rendered on a
new line instead of overwriting the current line.

This should result in human-readable output without any changes needed by
consumers, even when the system is writing to a non-TTY destination.

#### Manually Stepping Animation
If you'd like to manually animate the spinner, you can do so by setting the
`TerminalMode` to `ForceNoTTYMode | ForceSmartTerminalMode`. In this mode the
spinner will still use colors and other text stylings, but the animation only
happens when data is updated and on individual lines. You can accomplish this by
calling the `Message()` method with the same used previously.

## Usage
```
go get github.com/theckman/yacspin
```

Within the `yacspin` package there are some default spinners stored in the
`yacspin.CharSets` variable, and you can also provide your own. There is also a
list of known colors in the `yacspin.ValidColors` variable.

### Example

There are runnable examples in the [examples/](https://github.com/theckman/yacspin/tree/master/examples)
directory, with one simple example and one more advanced one. Here is a quick
snippet showing usage from a very high level, with error handling omitted:

```Go
cfg := yacspin.Config{
	Frequency:       100 * time.Millisecond,
	CharSet:         yacspin.CharSets[59],
	Suffix:          " backing up database to S3",
	SuffixAutoColon: true,
	Message:         "exporting data",
	StopCharacter:   "✓",
	StopColors:      []string{"fgGreen"},
}

spinner, err := yacspin.New(cfg)
// handle the error

err = spinner.Start()

// doing some work
time.Sleep(2 * time.Second)

spinner.Message("uploading data")

// upload...
time.Sleep(2 * time.Second)

err = spinner.Stop()
```

## Spinners

The spinner animations below are recorded at a refresh frequency of 200ms. Some
animations may look better at a different speed, so play around with the
frequency until you find a value you find aesthetically pleasing.

yacspin.CharSets index | sample gif (Frequency: 200ms)
-----------------------|------------------------------
0 | ![0 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/0.gif)
1 | ![1 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/1.gif)
2 | ![2 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/2.gif)
3 | ![3 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/3.gif)
4 | ![4 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/4.gif)
5 | ![5 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/5.gif)
6 | ![6 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/6.gif)
7 | ![7 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/7.gif)
8 | ![8 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/8.gif)
9 | ![9 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/9.gif)
10 | ![10 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/10.gif)
11 | ![11 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/11.gif)
12 | ![12 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/12.gif)
13 | ![13 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/13.gif)
14 | ![14 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/14.gif)
15 | ![15 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/15.gif)
16 | ![16 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/16.gif)
17 | ![17 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/17.gif)
18 | ![18 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/18.gif)
19 | ![19 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/19.gif)
20 | ![20 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/20.gif)
21 | ![21 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/21.gif)
22 | ![22 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/22.gif)
23 | ![23 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/23.gif)
24 | ![24 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/24.gif)
25 | ![25 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/25.gif)
26 | ![26 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/26.gif)
27 | ![27 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/27.gif)
28 | ![28 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/28.gif)
29 | ![29 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/29.gif)
30 | ![30 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/30.gif)
31 | ![31 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/31.gif)
32 | ![32 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/32.gif)
33 | ![33 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/33.gif)
34 | ![34 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/34.gif)
35 | ![35 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/35.gif)
36 | ![36 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/36.gif)
37 | ![37 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/37.gif)
38 | ![38 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/38.gif)
39 | ![39 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/39.gif)
40 | ![40 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/40.gif)
41 | ![41 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/41.gif)
42 | ![42 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/42.gif)
43 | ![43 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/43.gif)
44 | ![44 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/44.gif)
45 | ![45 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/45.gif)
46 | ![46 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/46.gif)
47 | ![47 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/47.gif)
48 | ![48 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/48.gif)
49 | ![49 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/49.gif)
50 | ![50 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/50.gif)
51 | ![51 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/51.gif)
52 | ![52 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/52.gif)
53 | ![53 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/53.gif)
54 | ![54 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/54.gif)
55 | ![55 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/55.gif)
56 | ![56 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/56.gif)
57 | ![57 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/57.gif)
58 | ![58 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/58.gif)
59 | ![59 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/59.gif)
60 | ![60 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/60.gif)
61 | ![61 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/61.gif)
62 | ![62 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/62.gif)
63 | ![63 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/63.gif)
64 | ![64 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/64.gif)
65 | ![65 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/65.gif)
66 | ![66 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/66.gif)
67 | ![67 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/67.gif)
68 | ![68 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/68.gif)
69 | ![69 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/69.gif)
70 | ![70 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/70.gif)
71 | ![71 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/71.gif)
72 | ![72 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/72.gif)
73 | ![73 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/73.gif)
74 | ![74 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/74.gif)
75 | ![75 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/75.gif)
76 | ![76 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/76.gif)
77 | ![77 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/77.gif)
78 | ![78 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/78.gif)
79 | ![79 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/79.gif)
80 | ![80 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/80.gif)
81 | ![81 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/81.gif)
82 | ![82 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/82.gif)
83 | ![83 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/83.gif)
84 | ![84 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/84.gif)
85 | ![85 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/85.gif)
86 | ![86 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/86.gif)
87 | ![87 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/87.gif)
88 | ![88 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/88.gif)
89 | ![89 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/89.gif)
90 | ![90 gif](https://raw.githubusercontent.com/theckman/yacspin-gifs/11953a4f12560eaf4a27054d3adad471eb19193c/spinners/90.gif)
