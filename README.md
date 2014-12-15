# Overview
Our projects aims to identify and respond to port scans against one or more systems from one or more systems. The project implements a neural net designed to catch slower passive scans, fast aggressive scans, horizontal scans against many systems, and vertical scans against a single system using potentially spoofed addresses. Based on the results from the neural net, our system would alarm and activate other applications to protect the system against scans. Our projects aims to develops a detection system that identifies scans earlier, and detects stealthy scans that slip past existing port scan detection software.

# Process
![System architecture](https://docs.google.com/drawings/d/1RNIRRjpCY45OXpBxJLRGVe2dGiWd20gSED2H-Dn1qjk/pub?w=960&h=720)
Our system consists of four parts: the collection agent, the detection agent, the response agent, and a database. 
The collection agent captures all packets entering a system, and logs relevant information about the packets to the database. These pieces of information are:
- Destination port
- Destination address
- Source address
- Time arrived
- Time to live
- ~~Result of Snort check for malicious packets (assuming this does not slow down too far)~~

Once the collection agent has reached a threshold, either n packets entering the collection agent or n seconds passing by, the collection agent will create a set of unique source address from the packets it has collected. For each unique source address, it places a job in to the detection agent’s queue.

The detection agent pulls jobs off its queue, and pulls all related packets from the database for the source address. It then calculates several features about the packet data, and enters those features into the neural net for classification. These features are:

1. Seen subnet scan before
2. Number of irregular ports (not traditionally running services)
3. Average time between different ports
4. Number of ports in n seconds
5. Ratio of packets to number of ports
6. Average TTL of packets from same source
7. Difference between max and min TTL
8. ~~A bias on where the traffic came from geographically.~~(IP to geo currently not supported)
9. ~~Number of flagged packets~~ (Snort integration not supported)
10. ~~A count of the number of ports that the screening system found to be undesirable.  Malformed packets/empty payloads/etc.~~ (Snort integration not supported)

If the neural net detects a port scan, a job is placed on the response agent’s queue containing the source address of the attacker.

The response agent pulls jobs off its queue and logs the relevant information. It then activates any number of scripts to further protect the system, including blocking the scanning system or trying to identify further information about the attacker. Our system is designed to allow multiple response agents to be running in parallel, allowing the system to scale up to an attack load.

# Data
We used the DARPA Intrusion Detection Data Sets from 1999. (http://www.ll.mit.edu/mission/communications/cyber/CSTcorpora/ideval/data/)


# Evaluation
We measured effectiveness by doing 10-fold cross validation on the training data.

# Presentation
[Slides for the final presentation](https://docs.google.com/presentation/d/1th6rvQ79YW52-BZvkmWwdCs9tmTbjGukOMyATNWydDw/edit?usp=sharing)


# External Links

### Libraries:
http://www.pybrain.org/docs/

http://scikit-learn.org/stable/

http://www.rabbitmq.com/

### Data:
http://www.ll.mit.edu/mission/communications/cyber/CSTcorpora/ideval/data/1999data.html

### Papers:
https://media.blackhat.com/bh-us-10/whitepapers/Engebretson_Pauli_Cronin/BlackHat-USA-2010-Engebretson-Pauli-Cronin-SprayPAL-wp.pdf

http://www.dsu.edu/research/ia/documents/%5B15%5D-Attack-Traffic-Libraries-for-Testing-and-Teaching-Intrusion-Detection-Systems.pdf
