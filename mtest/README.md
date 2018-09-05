How to manually run CKE using placemat
======================================

1. Run `make setup`
2. Run `make placemat`
3. Login to `host1` by:

    ```console
    $ chmod 600 mtest_key
    $ ssh -i mtest_key cybozu@10.0.0.11
    ```

4. Run `/data/setup-cke.sh` on `host1`.
5. Run `cke` on `host1`.
6. Copy `/data/cluster.yml` to `$HOME`, edit the copy, and load it by:

    ```console
    $ /data/ckecli constraints set control-plane-count 3
    $ /data/ckecli cluster set $HOME/cluster.yml
    ```

7. To stop placemat, run `sudo pkill placemat`.
