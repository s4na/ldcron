class Ldcron < Formula
  desc "cron形式でlaunchdジョブを管理するmacOS CLIツール"
  homepage "https://github.com/s4na/ldcron"
  version "0.1.23"
  license "MIT"

  on_arm do
    url "https://github.com/s4na/ldcron/releases/download/v#{version}/ldcron-darwin-arm64.tar.gz"
    sha256 "7a62d7b92b8dde9116adbb04242ac4544b847fe7edb8a5addc2f256aacf77178"
  end

  on_intel do
    url "https://github.com/s4na/ldcron/releases/download/v#{version}/ldcron-darwin-amd64.tar.gz"
    sha256 "31318c257973f7248be0d05a69bddea7f04b923177966891c56b878b25b400e6"
  end

  def install
    on_arm do
      bin.install "ldcron-darwin-arm64" => "ldcron"
    end
    on_intel do
      bin.install "ldcron-darwin-amd64" => "ldcron"
    end
  end

  test do
    # Test that the binary runs and shows help
    assert_match "ldcron", shell_output("#{bin}/ldcron --help")
  end
end
