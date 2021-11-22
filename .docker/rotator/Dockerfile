ARG USER_NAME='app'
ARG USER_UID='1000'
ARG USER_GID='1000'
ARG USER_GROUP='app_users'
ARG REPO='github.com/kohirens/iam-user-key-rotator'
ARG GLIBC_VER='2.34-r0'

FROM golang:1.17-alpine3.14 AS dev

ARG USER_NAME
ARG USER_UID
ARG USER_GID
ARG USER_GROUP
ARG REPO
ARG GLIBC_VER

ENV GOPATH /home/${USER_NAME}
ENV CGO_ENABLED=0
ENV WORK_DIR=/home/${USER_NAME}/src/${REPO}
ENV SHELL=/bin/sh
ENV LD_LIBRARY_PATH="/usr/lib:/lib"

# VS Code Requirements: openssh, musl, libgcc, libstdc++
RUN apk --no-progress --purge --no-cache upgrade \
 && apk --no-progress --purge --no-cache add --upgrade \
    curl \
    expect \
    git \
    gnupg \
    jq \
    libgcc \
    libstdc++ \
    openssh \
 && apk --no-progress --purge --no-cache upgrade \
 && rm -vrf /var/cache/apk/* \
 && rm -rf /tmp/* \
 && curl --version \
 && git --version

# Install vanilla GLibC: https://github.com/sgerrand/alpine-pkg-glibc
RUN curl -o /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub \
 && curl -LO https://github.com/sgerrand/alpine-pkg-glibc/releases/download/${GLIBC_VER}/glibc-${GLIBC_VER}.apk \
 && apk add glibc-${GLIBC_VER}.apk \
 && rm glibc-${GLIBC_VER}.apk \
       /etc/apk/keys/sgerrand.rsa.pub

# Add a non-root group and user, helpful if you dev on Linux.
RUN addgroup --system --gid ${USER_GID} ${USER_GROUP} \
 && adduser --system \
    --disabled-password \
    --ingroup ${USER_GROUP} \
    --uid ${USER_UID} \
    ${USER_NAME}

USER ${USER_NAME}

# Make directories with correct permissions
RUN mkdir -p ~/bin

# Install Go helpful dev tools.
RUN go install github.com/git-chglog/git-chglog/cmd/git-chglog@v0.15.0 \
 && go install golang.org/x/tools/gopls@latest \
 && go clean -cache -modcache -testcache

COPY .devcontainer/download-vs-code-server.sh .

# Install VS Code Server and Requirements for pre-installing extensions
RUN ./download-vs-code-server.sh \
 && mkdir -p /home/${USER_NAME}/.vscode-server/extensions \
 && mkdir -p /home/${USER_NAME}/.vscode-server-insiders/extensions

ENV PATH="/home/${USER_NAME}/bin:${PATH}"

WORKDIR "/home/${USER_NAME}/src/${REPO}"

COPY --chown=app:app_users ./go.* ./
COPY --chown=app:app_users ./*.go ./

RUN go mod tidy

COPY .docker/rotator/start.sh /usr/local/bin/start.sh

ENTRYPOINT [ "sh" ]

CMD [ "/usr/local/bin/start.sh" ]

FROM dev AS build

ARG USER_NAME
ARG USER_UID
ARG USER_GID
ARG USER_GROUP
ARG REPO

COPY .. .

RUN go build && go install

FROM alpine:3.14 AS release

ARG USER_NAME
ARG USER_UID
ARG USER_GID
ARG USER_GROUP

RUN apk --no-progress --purge --no-cache upgrade \
 && apk --no-progress --purge --no-cache add --upgrade \
    bash \
    curl \
    git \
 && apk --no-progress --purge --no-cache upgrade \
 && rm -vrf /var/cache/apk/* \
 && rm -rf /tmp/*

# Add a non-root group and user, helpful if you dev on Linux.
RUN addgroup --system --gid ${USER_GID} ${USER_GROUP} \
 && adduser --system \
    --disabled-password \
    --ingroup ${USER_GROUP} \
    --uid ${USER_UID} \
    ${USER_NAME}

USER ${USER_NAME}

# Make directories with correct permissions
RUN mkdir -p ~/bin

COPY --from=build --chown=${USER_NAME}:${USER_GROUP} /home/${USER_NAME}/bin/iam-user-key-rotator /home/${USER_NAME}/bin
COPY --from=build --chown=${USER_NAME}:${USER_GROUP} /home/${USER_NAME}/bin/git-chglog /home/${USER_NAME}/bin

ENV PATH="/home/${USER_NAME}/bin:${PATH}"

ENTRYPOINT [ "iam-user-key-rotator" ]

CMD [ "" ]