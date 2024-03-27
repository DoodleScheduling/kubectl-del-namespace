# typed: false
# frozen_string_literal: true

# This file was generated by GoReleaser. DO NOT EDIT.
class KubectlDelNamespace < Formula
  desc "kubectl plugin to forcefully delete a kubernetes namespace"
  homepage "https://github.com/DoodleScheduling/kubectl-del-namespace"
  version "0.0.2"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/DoodleScheduling/kubectl-del-namespace/releases/download/v0.0.2/kubectl-del-namespace_0.0.2_darwin_amd64.tar.gz"
      sha256 "b06123c037356139fcd202924b45e02eb5dd98a61dc874a0112536ef3b9611d5"

      def install
        bin.install "kubectl-del-namespace"
      end
    end
    if Hardware::CPU.arm?
      url "https://github.com/DoodleScheduling/kubectl-del-namespace/releases/download/v0.0.2/kubectl-del-namespace_0.0.2_darwin_arm64.tar.gz"
      sha256 "481dffd2b2512c6d6ee1e014c123a2926b9257858edccafdb11b0e3638b07a64"

      def install
        bin.install "kubectl-del-namespace"
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/DoodleScheduling/kubectl-del-namespace/releases/download/v0.0.2/kubectl-del-namespace_0.0.2_linux_arm64.tar.gz"
      sha256 "0a22efe656ffecb9f2ab109a839bc4124ed1483266acc8c6cf759390bc6786f0"

      def install
        bin.install "kubectl-del-namespace"
      end
    end
    if Hardware::CPU.intel?
      url "https://github.com/DoodleScheduling/kubectl-del-namespace/releases/download/v0.0.2/kubectl-del-namespace_0.0.2_linux_amd64.tar.gz"
      sha256 "f9d88cd31ee6bf5cc690a722b8823414322a54a3cbd1f7b8426258159b8fa631"

      def install
        bin.install "kubectl-del-namespace"
      end
    end
  end

  test do
    system "#{bin}/kubectl-del-namespace -h"
  end
end
