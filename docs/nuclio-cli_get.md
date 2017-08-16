## nuclio-cli get

Display one or many resources

### Synopsis


Display one or many resources

### Options

```
      --all-namespaces   Show resources from all namespaces
  -h, --help             help for get
  -l, --labels string    Label selector (lbl1=val1,lbl2=val2..)
  -o, --output string    Output format - text|wide|yaml|json (default "text")
  -w, --watch            Watch for changes
```

### Options inherited from parent commands

```
  -k, --kubeconfig string   Path to Kubernetes config (admin.conf) (default "C:\\Users\\yaron\\.kube\\config")
  -n, --namespace string    Kubernetes namespace (default "default")
  -v, --verbose             verbose output
```

### SEE ALSO
* [nuclio-cli](nuclio-cli.md)	 - nuclio command line interface
* [nuclio-cli get function](nuclio-cli_get_function.md)	 - Display one or many functions

###### Auto generated by spf13/cobra on 17-Aug-2017