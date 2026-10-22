{ pkgs, lib, config, inputs, ... }:

{
  dotenv.enable = true;

  # Tools the reviewer's passes shell out to (see docs: /appendix/existing-tools).
  packages = [
    pkgs.lychee # deterministic internal/external link + anchor checker (never an LLM)
    pkgs.vale # prose / acronym linter for the content pass
    pkgs.just # task runner (mirrors the justfile)
    pkgs.lsof # port preflight
    pkgs.librsvg
  ];

  # Node 22 powers the docs site (Rspress, .ts config + .tsx components).
  languages.javascript = {
    enable = true;
    package = pkgs.nodejs_22;
  };
  languages.typescript.enable = true;

  # The reviewer itself is written in Go: static types and explicit errors are
  # the constraints that keep an LLM-assisted codebase maintainable (see ADR 0001).
  languages.go.enable = true;

  enterShell = ''
    echo "llm-site-reviewer devenv"
    echo "  node    $(node --version)"
    echo "  go      $(go version 2>/dev/null | cut -d' ' -f3)"
    echo "  lychee  $(lychee --version 2>/dev/null | head -1)"
    echo "  vale    $(vale --version 2>/dev/null)"
    echo "  just    $(just --version)"
  '';

  # `devenv test` — smoke that the toolchain is actually present.
  enterTest = ''
    node --version
    lychee --version
    vale --version
  '';
}
