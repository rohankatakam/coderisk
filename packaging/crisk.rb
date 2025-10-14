# Homebrew Formula for CodeRisk CLI
# This file will be automatically updated by GoReleaser
# Documentation: https://docs.brew.sh/Formula-Cookbook

class Crisk < Formula
  desc "Lightning-fast AI-powered code risk assessment"
  homepage "https://coderisk.dev"
  version "1.0.0"  # Will be updated by GoReleaser
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/rohankatakam/coderisk-go/releases/download/v1.0.0/crisk_darwin_arm64.tar.gz"
      sha256 "CHECKSUM_WILL_BE_UPDATED_BY_GORELEASER"  # GoReleaser auto-updates
    else
      url "https://github.com/rohankatakam/coderisk-go/releases/download/v1.0.0/crisk_darwin_x86_64.tar.gz"
      sha256 "CHECKSUM_WILL_BE_UPDATED_BY_GORELEASER"  # GoReleaser auto-updates
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/rohankatakam/coderisk-go/releases/download/v1.0.0/crisk_linux_arm64.tar.gz"
      sha256 "CHECKSUM_WILL_BE_UPDATED_BY_GORELEASER"  # GoReleaser auto-updates
    else
      url "https://github.com/rohankatakam/coderisk-go/releases/download/v1.0.0/crisk_linux_x86_64.tar.gz"
      sha256 "CHECKSUM_WILL_BE_UPDATED_BY_GORELEASER"  # GoReleaser auto-updates
    end
  end

  def install
    bin.install "crisk"
  end

  test do
    system "#{bin}/crisk", "--version"
  end

  def caveats
    <<~EOS
      CodeRisk has been installed successfully!

      Quick Start:
      1. Configure your OpenAI API key (REQUIRED):
         $ export OPENAI_API_KEY="sk-..."
         # Or use: crisk configure

      2. Start infrastructure (REQUIRED - one-time per repo):
         $ docker compose up -d

      3. Initialize repository (REQUIRED - 10-15 min one-time):
         $ cd your-repo
         $ crisk init-local

      4. Check for risks (2-5 seconds):
         $ crisk check

      Setup time: ~17 minutes one-time per repo
      Check time: 2-5 seconds (after setup)
      Cost: $0.03-0.05/check (BYOK)

      Learn more: https://docs.coderisk.dev
    EOS
  end
end
