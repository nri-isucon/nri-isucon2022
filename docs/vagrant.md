# Vagrantで競技を体験する

競技VMをVagrantで起動して、実際の競技を体験することができます。

初期設定では、vCPU=1/Memory=2048のVMが1台起動します。コンテスト実施時のスペックは[README.md](../README.md)に記載されています。実行環境のリソースに余裕があれば、構成ファイル(Vagrantfile)を変更して、コンテスト実施時の環境に近づけることが可能です。

## 競技に必要な環境

- インターネットに接続できる、以下のソフトウェアが動作するコンピュータ
    - VirtualBox (https://www.virtualbox.org/)
    - Vagrant (https://www.vagrantup.com/)

## VM起動

競技VMをVagrantコマンドを使って起動します。

```shell
(nri-isucon2022)$ cd provisioning
(nri-isucon2022/provisioning)$ vagrant up

...(省略)...

PLAY RECAP *********************************************************************
default                    : ok=57   changed=51   unreachable=0    failed=0    skipped=4    rescued=0    ignored=0
```

初回起動時にAnsibleによるプロビジョニングが実行されます。スペックにより、数分から数十分間の時間がかかります。
プロビジョニング完了後、以下を確認して、正常にプロビジョニングが完了したことを確認してください。

1. `vagrant up`のAnsibleの結果がfailedが0(`failed=0`)であること
1. ホストPCのブラウザから`http://localhost:8080/`にアクセスし、Webアプリケーションが表示されること

## 競技方法

レギュレーションおよび当日マニュアルに従って、Webアプリケーションの改善を行います。

- [レギュレーション](./regulation.md)
- [当日マニュアル](./manual.md)

なお、実際のコンテスト時の環境との差異によって、説明が適切ではない場合があります。以降の記載も参考に競技を楽しんでください。

### サーバへのログイン方法

VMへは、以下のコマンドでログインできます。競技は`isucon`ユーザで行います。

```shell
$ vagrant ssh
$ sudo su - isucon
```

### ベンチマークの実行方法

ベンチマークは、以下のコマンドで実行できます。

```shell
$ TARGET_PORT=80 TARGET_IP_ADDR=127.0.0.1 DATA_PATH=/home/isucon/isubnb/bench/initial-data /home/isucon/isubnb/bench/benchmarker
```

ベンチマークの実行結果のサンプルは以下のとおりです。

```shell
---Benchmark Result---
{"pass":true,"failure_reason":"","language":"go","score":3064,"success_count":1038,"failure_count":1,"errors_dict":{}}
```
