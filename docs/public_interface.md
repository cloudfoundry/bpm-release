# Public Interface

BPM 1.0 carries with it guarantees about breaking backwards compatibility with
existing releases. However, not all of BPM is part of the public interface and
therefore should not be depended upon.

The following things you *can* depend on until BPM 2.0. If you see any of these
change before them then please file an issue so that we can address it.

* configuration file format

* existing `bpm` commands and their flags

* runtime environment (excluding bugs or security issues)

* pidfile path

* `bpm` executable path

* `bpm` job name and release name

* log file paths

Even with these guarantees, all software is bound by [Hyrum's Law][hyrum]. If an
internal change does break you then we can help you move to supported
interfaces.

[hyrum]: http://www.hyrumslaw.com/
