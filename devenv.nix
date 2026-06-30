{ pkgs, lib, config, inputs, ... }:

{
  dotenv.enable = true;

  # Tools the reviewer's passes shell out to (see docs: /landscape/existing-tools).
  packages = [
    pkgs.lychee     # deterministic internal/external link + anchor checker (never an LLM)
    pkgs.vale       # prose / acronym linter for the content pass
    pkgs.chromium   # headful browser the visual pass drives
    pkgs.just       # task runner (mirrors the justfile)
    pkgs.lsof       # port preflight
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

  # Playwright on NixOS: never let it download its own browsers (the prebuilt binaries
  # won't run against the Nix dynamic linker). Point it at the Nix-built browsers instead.
  env.PLAYWRIGHT_BROWSERS_PATH = pkgs.playwright-driver.browsers;
  env.PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD = "1";
  env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH = "${pkgs.chromium}/bin/chromium";

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
    chromium --version
  '';
}
