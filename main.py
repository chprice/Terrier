import time
from neural_net import Connection, Aggregator, NeuralNet

server = 'postgresql://terriergen:avahi-daemon@172.16.10.67:5432/terriergen'
s1 = time.time()
c = Connection('postgresql', 'terriergen', 'avahi-daemon', '172.16.10.67', 5432, 'terriergen')
#p= c.getPackets('192.168.1.10')
#ports = [x.port for x in p]
#print(set(ports))
#print(len(p))
#s2 = time.time()
a = Aggregator()
#traits = a.aggregate(p)
#print(traits)
#s3 = time.time()

#print(s2-s1)
#print(s3-s2)
#print(s3-s1)
               
nn = NeuralNet(a, c)
nn.crossValidation("out.json")
