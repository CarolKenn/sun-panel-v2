# build frontend
# 与 CI 工作流保持一致：工作流用 Node 22.10.0 + npm 验证过构建，这里复用同一个版本
ARG NODE_VERSION=22
FROM node:${NODE_VERSION}-alpine AS web_image

# 使用淘宝npm镜像源加速依赖安装
RUN npm config set registry https://registry.npmmirror.com

WORKDIR /build

# 先复制整个项目（排除.dockerignore中的文件）
COPY . /build

# 安装依赖
# 依赖安装统一走 npm ci（依据 package-lock.json）：CI、Dockerfile、build.sh 三处一致。
# 仓库不再保留 pnpm-lock.yaml——它曾丢失 @vicons/ionicons5 已与 package.json 失同步，
# 任何人照它跑 --frozen-lockfile 都会直接失败。
RUN npm ci --no-audit --no-fund

# 构建项目
RUN npm run build

# build backend
# 最新alpine3.19导致sqlite3编译失败(https://github.com/mattn/go-sqlite3/issues/1164，
# 临时解决方案:https://github.com/mattn/go-sqlite3/pull/1177)
# sun-panel暂时解决方案使用golang:1.21-alpine3.18（因旧版本使用没问题，短期内较稳定） 
FROM golang:1.21-alpine3.18 AS server_image

WORKDIR /build

COPY ./service .

# 中国国内源
# RUN sed -i "s@dl-cdn.alpinelinux.org@mirrors.aliyun.com@g" /etc/apk/repositories \
#     && go env -w GOPROXY=https://goproxy.cn,direct

RUN apk add --no-cache bash curl gcc git musl-dev

# 生成嵌入资源时注意两点：
# 1. 先删旧的 assets/bindata.go，否则它会被自己再扫一遍，包体积逐次递增
# 2. go-bindata-assetfs 会在 CWD 额外落一个没有 package 声明的 bindata.go
#    （assetfs 辅助片段，项目只用到 assets.Asset），留在源码树里会让
#    go build ./... 报 expected 'package', found 'func'
RUN go env -w GO111MODULE=on \
    && export PATH=$PATH:/go/bin \
    && go install -a -v github.com/go-bindata/go-bindata/...@latest \
    && go install -a -v github.com/elazarl/go-bindata-assetfs/...@latest \
    && rm -f assets/bindata.go \
    && go-bindata-assetfs -o=assets/bindata.go -pkg=assets assets/... \
    && rm -f bindata.go \
    && go build -o sun-panel --ldflags="-X sun-panel/global.RUNCODE=release -X sun-panel/global.ISDOCKER=docker" main.go



# run_image
FROM alpine

WORKDIR /app

COPY --from=web_image /build/dist /app/web

COPY --from=server_image /build/sun-panel /app/sun-panel

# 中国国内源
# RUN sed -i "s@dl-cdn.alpinelinux.org@mirrors.aliyun.com@g" /etc/apk/repositories

EXPOSE 3002

RUN apk add --no-cache bash ca-certificates su-exec tzdata \
    && chmod +x ./sun-panel \
    && ./sun-panel -config

CMD ["./sun-panel"]
