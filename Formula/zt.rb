class Zt < Formula
  desc "Zero Trust tunnel manager for Cloudflare"
  homepage "https://github.com/casablanque-code/cfzt"
  version "0.11.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.11.0/zt-darwin-arm64"
      sha256 "3b5de81ea89c5237d8d6738a634b60dfdd412e8beef61507cee244f10ee7a239"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.11.0/zt-darwin-amd64"
      sha256 "ebfdac93f8b0e698fe5d68dfda713ba2e768443328e5f2456749c18c788f5a5d"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.11.0/zt-linux-arm64"
      sha256 "288bff157a8d36541a936b80f485581177b20ad1bba2483504a9c9af127e550c"
    else
      url "https://github.com/casablanque-code/cfzt/releases/download/v0.11.0/zt-linux-amd64"
      sha256 "58aa1cc7dbc0c3066956f38b08727a82245b52a7236705cded3f2c08875719fc"
    end
  end

  def install
    bin.install Dir["zt-*"].first => "zt"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/zt version")
  end
end
