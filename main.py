import time
from neural_net import Connection, Aggregator, NeuralNet

c = Connection('postgresql', 'terriergen', 'avahi-daemon', '172.16.10.67', 5432, 'terriergen')
a = Aggregator()
nn = NeuralNet(a, c)

#nn.crossValidation("out.json") # 82.88%

nn.trainFromFile("out.json")
print "Exporting to file"
nn.exportToFile("nn.trained")
