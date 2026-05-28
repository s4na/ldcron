class Ldcron < Formula
  desc "cron形式でlaunchdジョブを管理するmacOS CLIツール"
  homepage "https://github.com/s4na/ldcron"
  version "0.1.24"
  license "MIT"

  on_arm do
    url "https://github.com/s4na/ldcron/releases/download/v#{version}/ldcron-darwin-arm64.tar.gz"
    sha256 "ec3a919336c0c32d6861ffb713ec07044391ff5da3ab3ac4f326fa4e10de0299"
  end

  on_intel do
    url "https://github.com/s4na/ldcron/releases/download/v#{version}/ldcron-darwin-amd64.tar.gz"
    sha256 "f7b70474413099ac7645df2ea0b37ee767c7e5b4b8c0bbd4fb886874a2ab71a4"
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
