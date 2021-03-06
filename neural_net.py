# Channon Price
# Neural Network
# This file contains the code necessary to train a port scan detector neural net
#   and classify a set of packets given a fully trained neural net.
#http://www.pybrain.org/

import pickle
import json
from datetime import timedelta
from math import log

from pybrain.tools.shortcuts import buildNetwork
from pybrain.tools.validation import CrossValidator
from pybrain.structure import TanhLayer
from pybrain.supervised.trainers import BackpropTrainer
from pybrain.datasets import SupervisedDataSet

from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from sqlalchemy.engine.url import URL
from sqlalchemy import between

from db_schema import Packet

def ln(x):
    if(x==0):
        return 0
    return log(x)

# Wrapper around the pybrain Neural Net. Provides methods to import/export a neual net
#  and train or classify the neural net.
class NeuralNet:
    def __init__(self, aggregator, connection):
        self.aggregator = aggregator
        self.connection = connection
        self.net = buildNetwork(7, 21, 2, hiddenclass=TanhLayer) # This really needs to get evaluated
        #I'm also officially defining [1,0] as scan, [0,1] as clean. aka [scan, clean]
        
    #def __str__(self)
    #def __repr__(self)
    
    def importFromFile(self, filename):
        self.net = pickle.load(open(filename,'r'))
        
    def exportToFile(self, filename):
        pickle.dump(self.net, open(filename,'w'))

    def checkScan(self, ip_address):
        return self.classify(self.aggregator.aggregate(self.connection.getPackets(ip_address)))
    
    def classify(self, traits):
        output = self.net.activate(traits.values()) #I'm making the assumption that traits will always contain the same keys in the same order.
        if(output[0] > output[1]):
            return True # Scan occurred
        else:
            return False # No scan
        # It had abs(output[0] - output[1]) certainty.
        

    def createDataSetFromFile(self, filename):
        print "Creating data set from file"
        ds = SupervisedDataSet(7, 2)
        for loc, data in enumerate(json.loads(open(filename).readline())):
            if(loc%100==0):
                print loc, "training ip sets processed"
                
            trainingData = [self.aggregator.aggregate(self.connection.getPacketsBounded(data["Ip"], data["Start"], data["End"])), data["Scan"]]
            if(trainingData[1]):
                ds.addSample(trainingData[0].values(), (1,0))
            else:
                ds.addSample(trainingData[0].values(), (0,1))
        print "Data set created"
        return ds

    def trainFromFile(self, filename):  
        trainer = BackpropTrainer(self.net, self.createDataSetFromFile(filename))
        trainer.trainUntilConvergence(maxEpochs=10)
        
    def crossValidation(self, filename):
        trainer = BackpropTrainer(self.net)
        crossValidator = CrossValidator(trainer, self.createDataSetFromFile(filename), n_folds=10)
        result = crossValidator.validate()
        print result*100, "%"


# Aggregator serves as a utility class to help convert raw packet data into useful metrics for the neural net.
# It provides each metric as an individual function. To get all metrics, use the aggregate() method.
class Aggregator:
    #def __init__(self)
    def aggregate(self, packets):
        traits = dict()
        
        traits["seenSubnet"] = self.seenSubnet(packets)
        traits["numberIrregularPorts"] = ln(self.numberIrregularPorts(packets))
        traits["averageTimeBetweenPorts"] = self.averageTimeBetweenPorts(packets)
        traits["numberPorts"] = ln(self.numberPorts(packets))
        traits["ratioPacketsToPorts"] = ln(self.ratioPacketsToPorts(packets))
        traits["averageTTL"] = ln(self.averageTTL(packets))
        traits["diffTTL"] = self.diffTTL(packets)

        return traits


    def seenSubnet(self, packets):
        return 0 # TODO (connect to the database, update seen table)
    
    def numberIrregularPorts(self, packets):
        regular_ports = [1, 5, 7, 18, 20, 21, 22, 23, 25, 29, 37, 42, 43, 49, 53, 69, 70, 79, 80, 103, 108, 109, 110, 115, 118, 119, 137, 139, 143, 150, 156, 161, 179, 190, 194, 197, 389, 396, 443, 444, 445, 458, 546, 547, 563, 569, 1080] # http://www.webopedia.com/quick_ref/portnumbers.asp
        seen_irr_ports = [packet.port for packet in packets if packet.port not in regular_ports]
        return len(set(seen_irr_ports))

    def averageTimeBetweenPorts(self, packets): # TODO
        #times = sorted([packet.time for packet in packets]) # Assume packets are in order already
        seenPort = packets[0].port
        seenTime = packets[0].time
        totalTime = timedelta()
        totalSegments = 1
        for packet in packets:
            if seenPort != packet.port:
                totalTime += packet.time - seenTime
                totalSegments += 1
                seenPort = packet.port
                seenTime = packet.time
        if(totalSegments == 1):
            totalTime = packets[-1].time - packets[0].time
            
        return float(totalTime.total_seconds())/totalSegments
        
    def numberPorts(self, packets):
        return len(set([packet.port for packet in packets]))
    
    def ratioPacketsToPorts(self, packets):
        return (1.0*len(packets))/len(set([packet.port for packet in packets]))
    
    def averageTTL(self, packets):
        return sum([packet.ttl for packet in packets])//(1.0*len(packets))

    def diffTTL(self, packets):
        ttls = [packet.ttl for packet in packets]
        return max(ttls) - min(ttls)

# Connection wraps the sqlalchemy connection and provides methods to fetch packets.
class Connection:
    def __init__(self, drivername, db_user, db_password, db_host, db_port, table_name):
        url = URL(drivername, username=db_user, password=db_password, host=db_host, port=db_port, database=table_name)
        Session = sessionmaker(bind=create_engine(url))
        self.session = Session()
        
    def getPackets(self, ip_address):
        return self.session.query(Packet).filter_by(ip=ip_address).order_by(Packet.time).all()
    
    def getPacketsBounded(self, ip_address, start_id, end_id):
        return self.session.query(Packet).filter(Packet.ip == ip_address, Packet.id >= start_id, Packet.id <= end_id).order_by(Packet.time).all()
