FROM gcr.io/distroless/static:nonroot@sha256:91ca4720011393f4d4cab3a01fa5814ee2714b7d40e6c74f2505f74168398ca9
WORKDIR /
COPY kubectl-del_namespace kubectl-del_namespace

ENTRYPOINT ["/kubectl-del_namespace"]
