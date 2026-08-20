# clippy
Clippy is your friend.  A very helpful friend.

## Usage

```
clippy "My Text Here"
```

This outputs a timestamped PNG in the current directory and prints its name.
Words may be quoted as one argument or supplied separately.

### Scaling Options

```
clippy -scale 0.5 "My Text Here"   # Half size output
clippy -scale 2.0 "My Text Here"   # Double size output
clippy -width 800 "My Text Here"   # 800px wide, height proportional
clippy -output message.png "Hello" # Choose an output file
clippy -output - "Hello" > message.png
```

### Text Options

The speech bubble automatically grows and shrinks around the message. Text
wraps at `-text-width`, and oversized messages shrink to fit `-text-height`.

```
clippy -font-size 96 -text-color '#336699' "Hello"
clippy -align center -padding 20 "Centered text"
clippy -text-x 100 -text-y 100 -text-width 1200 -text-height 400 "Custom box"
```

Run `clippy -help` for all available options.

## Building

* Clone repo
* `go build`
* `go install`

Requires Go 1.21 or later.

## Licenses
MIT License.

* Clippy is probably a registered trademark for Microsoft.
* Comic Sans MS is probably owned by Microsoft too.

This work is not affiliated with, or endorsed by Microsoft.
