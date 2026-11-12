class Ldcron < Formula
  desc "cron形式でlaunchdジョブを管理するmacOS CLIツール"
  homepage "https://github.com/s4na/ldcron"
  version "0.1.29"
  license "MIT"

  on_arm do
    url "https://github.com/s4na/ldcron/releases/download/v#{version}/ldcron-darwin-arm64.tar.gz"
    sha256 "0632a440d34a9e50ddae3a904ccfade8e556f53c01f63ad9567a67b1e853ecdf"
  end

  on_intel do
    url "https://github.com/s4na/ldcron/releases/download/v#{version}/ldcron-darwin-amd64.tar.gz"
    sha256 "700a098643e5147e4bc56f844d6cdfd25468ff4c4523f4bac5e11bb0de3991a5"
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
