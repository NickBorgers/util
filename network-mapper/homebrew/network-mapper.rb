class NetworkMapper < Formula
  desc "Cross-platform network discovery and visualization tool"
  homepage "https://github.com/NickBorgers/util"
  version "1.0.0-smart-crop"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/NickBorgers/util/releases/download/v#{version}/network-mapper-darwin-arm64.tar.gz"
      sha256 "baa4f84b0bee238bf66d33f9398fe735dd0b2d86dadaa1064cc0c794a169abc6"
    else
      url "https://github.com/NickBorgers/util/releases/download/v#{version}/network-mapper-darwin-amd64.tar.gz"
      sha256 "cc49943055121c553bd8829f39315053889318888d2eb04e61b96f4532917eed"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/NickBorgers/util/releases/download/v#{version}/network-mapper-linux-arm64.tar.gz"
      sha256 "00f89261e8a7ce47d6bb1e154ae68a0069348e0dc0a4278ed96b4bd5b707d743"
    else
      url "https://github.com/NickBorgers/util/releases/download/v#{version}/network-mapper-linux-amd64.tar.gz"
      sha256 "950e0af036251c861fbf4c42e5bc74cbd478794788bc5403eb55e5b1f3a557f4"
    end
  end

  def install
    bin.install "network-mapper-#{OS.kernel_name.downcase}-#{Hardware::CPU.arch}"
    mv bin/"network-mapper-#{OS.kernel_name.downcase}-#{Hardware::CPU.arch}", bin/"network-mapper"
  end

  def caveats
    <<~EOS
      Network Mapper may require elevated privileges for some operations:
      - On macOS/Linux: Run with 'sudo' if needed for network interface access
      - Some features like ARP scanning require root access

      To start with intelligent discovery (default, recommended):
        network-mapper

      To adjust thoroughness for intelligent discovery:
        network-mapper --thoroughness 1  # Minimal, faster
        network-mapper --thoroughness 5  # Exhaustive, thorough

      To run a quick brute-force scan (interface subnets only):
        network-mapper --scan-mode quick

      For help:
        network-mapper --help
    EOS
  end

  test do
    assert_match "Network Mapper", shell_output("#{bin}/network-mapper --version")
  end
end