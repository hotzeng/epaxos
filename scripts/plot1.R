library(tidyr);
library(dplyr);
library(ggplot2);

data <- read.csv(file='report/plot1.csv', header=FALSE, sep=',');
colnames(data) <- c('servers', 'records', 'pipe', 'collision', 'delay0', 'delayD', 'jitter', 'id', 'time');
data <- data %>%
	mutate(collision=collision / 100, jitter=jitter / 100) %>%
	mutate(delay=delay0 + delayD * id) %>%
	mutate(delayF=factor(delay)) %>%
	select(-servers, -records, -delay0, -delayD, -id);

pdf('report/plot1.pdf', 5, 5);

ggplot(data, aes(x=delayF, y=time)) +
	geom_violin(scale='area') +
	xlab('Server delay (ms)') +
	ylab('Client delay (ms)');

dev.off();
