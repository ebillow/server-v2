#!/bin/bash

# shellcheck disable=SC2164
# shellcheck disable=SC2046
# shellcheck disable=SC2086
# shellcheck disable=SC2103
# shellcheck disable=SC2155

cd $(dirname "$0")

# color
RED=""
GREEN=""
YELLOW=""
NC=""

if [ -z "${NO_COLOR}" ]; then
  RED="\033[0;31m"
  GREEN="\033[0;32m"
  YELLOW="\033[1;33m"
  NC="\033[0m"
fi

# 根据不同的平台，使用不同的 protoc 和 grpc 插件
BINDIR=""
case "$(uname -s)" in
Linux)
  BINDIR=./platform/linux
  ;;
Darwin)
  BINDIR=./platform/darwin
  ;;
*)
  BINDIR=./platform/win
  ;;
esac

# 将 platform 里的插件和工具添加到 PATH 里
export PATH=$(realpath ${BINDIR}):${PATH}
# echo $PATH

#echo -e "${YELLOW}打印路径${NC}"
#echo "BINDIR:             ${BINDIR}"
#echo "protoc:             $(which protoc)"
#echo "protoc-gen-go:      $(which protoc-gen-go)"
#echo "protoc-gen-go-grpc: $(which protoc-gen-go-grpc)"

# 打印工具和插件版本
echo ""
echo -e "${YELLOW}打印版本${NC}"
echo "protoc:             $(protoc --version)"
echo "protoc-gen-go:      $(protoc-gen-go --version)"
echo "protoc-gen-go-grpc: $(protoc-gen-go-grpc --version)"

trap 'echo -e "${RED}error: Script failed: see failed command above${NC}"' ERR

# 生成的目录
OUTDIR=$(realpath "../../pkg")
echo ${OUTDIR}

function gen_proto() {
  # proto 列表, 一行一个, 可以写注释
  protoList=(
    msg_common.proto
    msg.proto
    msg_base.proto
    msg_player.proto
    msg_server.proto
  )
  protoc --go_out=${OUTDIR} --plugin= -I ${OUTDIR}/proto "${protoList[@]}"
}

function gen_msg_id() {
     # 生成 msg id
      msgIds=(
        msg_id_c2s.proto
        msg_id_s2c.proto
        msg_id_s2s.proto
      )
      protoc --go_out=${OUTDIR}/pb -I ${OUTDIR}/proto "${msgIds[@]}"
}

# 生成grpc
function gen_grpc() {
  # proto 列表, 一行一个, 可以写注释
  protoList=(
    service.proto
  )
  protoc --go_out=${OUTDIR} --go-grpc_out=${OUTDIR} -I ${OUTDIR}/proto "${protoList[@]}"
}

# 生成注册消息
function gen_register() {
  go generate ./generate_msgid.go
  mv ./msg_id_c2s.proto "${OUTDIR}/proto/"
  mv ./msg_id_s2c.proto "${OUTDIR}/proto/"
  mv ./msg_id_s2s.proto "${OUTDIR}/proto/"
  echo "🎉 msg id generated!"

    go generate ./generate_proto.go
  mv ./msg_helper.go "${OUTDIR}/pb/"
  echo "🎉 msg helper generated!"
}

echo ""
echo -e "${GREEN}开始生成协议代码${NC}"

rm -rf ${OUTDIR}/pb/*.pb.go # 先删除原有的
echo "🎉 删除 .pb.go"
gen_proto
echo "🎉 生成proto"
gen_grpc
echo "🎉 生成grpc"
gen_register
echo "🎉 生成helper"
gen_msg_id
echo "🎉 生成msg id"

echo -e "${GREEN}生成协议完成${NC}"
