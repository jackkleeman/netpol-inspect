# netpol-inspect
A cli tool to predict and describe the effects of Kubernetes network policies

Kubernetes network policies can be non-linear in their impact. A given policy might add new allowed traffic routes for
some pods, and remove them for others. This is because if a pod doesn't already have any ingress policies applicable,
then it allows all ingress traffic, and applying a new ingress policy to it will therefore isolate the pod against a new
whitelist. It's not always obvious what policies you have running that could lead to this behaviour.

With netpol-inspect, you can predict the effects of a new policy, or describe the effects of an existing policies. It will
return a list of pods that have extra permissions as a result of the policy, and a list that are isolated in some way by it.

It can be useful to:
1. Figure out if its safe to delete a network policy; what pods are affected and are they more or less isolated?
2. Predict the impact of a policy that affects many pods. Will this be the first policy for some of those pods, isolating them?
3. Ensure that a policy correctly only affects the pods it's supposed to.

## Usage

```
$ netpol-inspect -n kube-system describe policy-name
May be allowed new ingress:
  [s-foo-abcd]

$ netpol-inspect apply -f netpol.yaml
Would allow all egress if not for this whitelist, may be allowed new ingress:
  [s-bar-efgh]
```
