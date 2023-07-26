# Glone

Glone is git clone for when you don't care about the git.

Let's say you want to download all of the shells from [nixpkgs](https://github.com/NixOS/nixpkgs). 
You could:
  - Wait for over five minutes to let the entire thing download, only to delete 99% of it. 
  - Manually click through every folder and file and go to the raw version of it, then sort through your downloads.
  - Use glone and download only the shells folder in a little under a second.

```glone https://github.com/NixOS/nixpkgs pkgs/shells```

In this scenario, glone isn't downloading the whole repository, instead it is asynchronously downloading each individual file.
While this is what allows glone to get such a fast result in this case, it can also lead to download times that are actually slower if the part of the repository that you're downloading is big enough. 
In these cases, you would probably want to use -t or --tar, to make glone download the entire tar.gz file, but only extract what is necessary.

But maybe you don't want to download all of the shells, you don't really care about downloading bash.
Simply use -F or --filter.

```glone https://github.com/NixOS/nixpkgs pkgs/shells -F bash```

(Filtering also supports multiple files and regex)

You can also use glone as a glorified wget/curl by passing in the -f or --file command.

```glone https://github.com/NixOS/nixpkgs -f pkgs/shells/dash/default.nix pkgs/shells/fish/default.nix pkgs/shells/elvish/default.nix```

## Installation
```go install github.com/brewingweasel/glone@latest```
