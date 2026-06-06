class Ldcron < Formula
  desc "cron形式でlaunchdジョブを管理するmacOS CLIツール"
  homepage "https://github.com/s4na/ldcron"
  version "0.1.28"
  license "MIT"

  on_arm do
    url "https://github.com/s4na/ldcron/releases/download/v#{version}/ldcron-darwin-arm64.tar.gz"
    sha256 "07fe0631872b1d3284f68ad8c5c762567b7076166cef93ed43ba1f82610e93fc"
  end

  on_intel do
    url "https://github.com/s4na/ldcron/releases/download/v#{version}/ldcron-darwin-amd64.tar.gz"
    sha256 "83c01163f07b625d2652f454d3b7151da38a9ccc6dc12106bff21a33250d3e99"
  end

  def install
    on_arm do
      bin.install "ldcron-darwin-arm64" => "ldcron"
    end
    on_intel do
      bin.install "ldcron-darwin-amd64" => "ldcron"
    end
  end

  def post_install
    return unless quiet_system "#{bin}/ldcron", "migrate", "--help"

    system "#{bin}/ldcron", "migrate", "--quiet"
  rescue RuntimeError => e
    opoo "ldcron plist migration did not complete: #{e.message}"
    opoo "Run `ldcron migrate` to retry."
  end

  test do
    # Test that the binary runs and shows help
    assert_match "ldcron", shell_output("#{bin}/ldcron --help")
  end
end
