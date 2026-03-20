
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

echo -e "${YELLOW}打印路径${NC}"
echo "BINDIR:             ${BINDIR}"
echo "protoc:             $(which protoc)"
echo "protoc-gen-go:      $(which protoc-gen-go)"
echo "protoc-gen-go-grpc: $(which protoc-gen-go-grpc)"

# 打印工具和插件版本
echo ""
echo -e "${YELLOW}打印版本${NC}"
echo "protoc:             $(protoc --version)"
echo "protoc-gen-go:      $(protoc-gen-go --version)"
echo "protoc-gen-go-grpc: $(protoc-gen-go-grpc --version)"

trap 'echo -e "${RED}error: Script failed: see failed command above${NC}"' ERR


# 生成的目录
OUTDIR=$(realpath "./target/table")
function gen_proto() {
  # proto 列表, 一行一个, 可以写注释
  protoList=(
    cfg.proto
  )
  protoc --go_out=${OUTDIR} --plugin= -I . "${protoList[@]}"
}

# 生成注册消息
function gen_register() {
  go generate ./main.go
  echo "🎉 generated!"
}

echo ""
echo -e "${GREEN}开始生成协议代码${NC}"

rm -rf ${OUTDIR}/*.pb.go # 先删除原有的

gen_register
gen_proto
# gen_cpp_tool

#echo ""
#echo -e "${YELLOW}按任意键继续${NC}"
#read -rn 1