## Toy example of the Usage of Kubernetes Controller ##

### Overview ###
As shown in the example about [Kubernetes Reflector](https://github.com/songbinliu/KubeReflectorTest), *Reflector* is a framwork of *producer*: it collects
the changes (or *Events*) of the Kubernetes resources. **Kubernetes Controller** providing a *Producer-Consumer* framework. It uses *Reflector*
to *produce* contents, and allow users to provide *EventHandlerFunc* to consume the contents.

### Definition of Controller ###

<img width="857" alt="controller" src="https://cloud.githubusercontent.com/assets/27221807/26830654/8ad7756c-4a97-11e7-9fe4-48b1e288a729.png">

Because *Events* come in some order, the storage for the *Reflector* should be able to keep this order; so it is reasonable to store the *Events* in a *Queue*
which can provide *FIFO* access order. Users can provide different *ProcessFunc* to consume the *Events*.

<img width="857" alt="controller.run" src="https://cloud.githubusercontent.com/assets/27221807/26830756/ddead29e-4a97-11e7-9854-af619be1bf9d.png">

When *Controller.Run()* is called, a *Reflector* is created and get run in a seperated goroutine to *Produce* *Events*, 
and *Controller.processLoop()* is running to *Consume* the *Events*.

### Run the Example ###
```console
go build

./KubeControllerTest --masterUrl $kubeMaster \ 
                     --namespace "default" \
                     --alsologtostderr
```
