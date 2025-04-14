class Imbued < Formula
  desc "Toolset for managing secrets in a development environment"
  homepage "https://github.com/novacove/imbued"
  url "https://github.com/novacove/imbued.git", tag: "v0.1.0"
  license "MIT"
  head "https://github.com/novacove/imbued.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "-o", bin/"imbued", "cmd/imbued/main.go"
    
    # Install shell scripts
    (share/"imbued/scripts/bash").install "scripts/bash/imbued.sh"
    (share/"imbued/scripts/fish").install "scripts/fish/imbued.fish"
    (share/"imbued/scripts/zsh").install "scripts/zsh/imbued.zsh"
    
    # Install macOS launchd files
    (share/"imbued/scripts/macos").install "scripts/macos/com.novacove.imbued.plist"
    (share/"imbued/scripts/macos").install "scripts/macos/install.sh"
  end

  def caveats
    <<~EOS
      To use imbued with bash, add the following to your .bashrc or .bash_profile:
        export IMBUED_BIN=#{opt_bin}/imbued
        source #{opt_share}/imbued/scripts/bash/imbued.sh

      To use imbued with zsh, add the following to your .zshrc:
        export IMBUED_BIN=#{opt_bin}/imbued
        source #{opt_share}/imbued/scripts/zsh/imbued.zsh

      To use imbued with fish, add the following to your config.fish:
        set -gx IMBUED_BIN #{opt_bin}/imbued
        source #{opt_share}/imbued/scripts/fish/imbued.fish

      To install imbued as a launchd service on macOS:
        mkdir -p ~/.imbued/logs
        sed "s|~|$HOME|g" #{opt_share}/imbued/scripts/macos/com.novacove.imbued.plist > ~/Library/LaunchAgents/com.novacove.imbued.plist
        launchctl load ~/Library/LaunchAgents/com.novacove.imbued.plist

      When using the launchd service, also set the socket path in your shell configuration:
        export IMBUED_SOCKET=$HOME/.imbued/imbued.sock  # for bash/zsh
        set -gx IMBUED_SOCKET $HOME/.imbued/imbued.sock  # for fish
    EOS
  end

  test do
    assert_match "Imbued", shell_output("#{bin}/imbued --version")
  end
end
