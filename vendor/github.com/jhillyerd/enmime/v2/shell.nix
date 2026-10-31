{ pkgs ? import <nixpkgs> { } }:
pkgs.mkShell {
  buildInputs = with pkgs; [
    delve
    go
    golangci-lint
    gopls
  ];

  hardeningDisable = [ "fortify" ];
}
