class Ldcron < Formula
  desc "cron形式でlaunchdジョブを管理するmacOS CLIツール"
  homepage "https://github.com/s4na/ldcron"
  version "0.1.22"
  license "MIT"

  on_arm do
    url "https://github.com/s4na/ldcron/releases/download/v#{version}/ldcron-darwin-arm64.tar.gz"
    sha256 "29b1b94f3def593575822cabb5275980d13f9c33e2260250a99ddeed213188ac"
  end

  on_intel do
    url "https://github.com/s4na/ldcron/releases/download/v#{version}/ldcron-darwin-amd64.tar.gz"
    sha256 "591f33bf7647c18796903f78962ad38d63c74dd5e3171af35ee8ea1e9a634b5a"
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
