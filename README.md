# clippy
Clippy is your friend.  A very helpful friend.

## Usage

```
clippy "My Text Here"
```

This will output a file in the current directory called `clippy{DATESTAMP}.png`

### Scaling Options

```
clippy -scale 0.5 "My Text Here"   # Half size output
clippy -scale 2.0 "My Text Here"   # Double size output
clippy -width 800 "My Text Here"   # 800px wide, height proportional
```

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
