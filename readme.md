实现了:
    megaraid里的获取控制器的信息、物理盘和逻辑盘信息, 方便实现信息获取和监控，常规获取信息可以脱离storcli64命令
    有些值未能找到文档对应关系，比如控制器的device interface = SAS-12G
参考了： github.com/dswarbrick/smart, 但是这个项目两年不维护了，没有更新megaraid相关的功能